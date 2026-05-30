package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"dnsherene/internal/report"
	"dnsherene/pkg/dnshe"
)

const (
	renewWindowDays           = 180
	rateLimitErrorCode        = "rate_limit_exceeded"
	maxRateLimitRetries       = 2
	maxRateLimitRetryInterval = 2 * time.Minute
)

// Result 表示一次续期任务的结构化结果。
type Result struct {
	Matched     int
	Renewed     int
	Failed      int
	DryRun      bool
	Domains     []report.DomainInfo
	RenewedList []report.RenewedDomain
	FailedList  []report.FailedDomain
}

// Service 负责续期任务编排：选目标、调用 SDK。
type Service struct {
	dnsClient *dnshe.Client
}

type renewFailuresError struct {
	Count   int
	Reasons []string
}

// Error 返回续期失败数量和摘要原因。
func (e *renewFailuresError) Error() string {
	if e == nil || e.Count <= 0 {
		return "renew requests failed"
	}

	message := fmt.Sprintf("%d renew request(s) failed", e.Count)
	if len(e.Reasons) > 0 {
		message += ": " + strings.Join(e.Reasons, ", ")
	}
	return message
}

// NewService 创建续期服务实例。
func NewService(dnsClient *dnshe.Client) (*Service, error) {
	if dnsClient == nil {
		return nil, fmt.Errorf("dns client is required")
	}
	return &Service{dnsClient: dnsClient}, nil
}

// Run 执行一次续期任务。
func (s *Service) Run(ctx context.Context, dryRun bool) (Result, error) {
	result := Result{}

	targets, err := s.listSubdomainsWithRateLimitRetry(ctx)
	if err != nil {
		return result, err
	}
	result.Domains = describeDomains(time.Now().UTC(), targets)

	targets = filterRenewableTargets(time.Now().UTC(), targets)
	if len(targets) == 0 {
		return result, nil
	}

	result.Matched = len(targets)
	result.DryRun = dryRun

	if dryRun {
		return result, nil
	}

	result.RenewedList = make([]report.RenewedDomain, 0, len(targets))
	result.FailedList = make([]report.FailedDomain, 0)
	failureReasons := make([]string, 0, 3)
	failureReasonSeen := make(map[string]struct{})

	for _, target := range targets {
		renewResult, renewErr := s.renewSubdomainWithRateLimitRetry(ctx, target.ID)
		if renewErr != nil {
			if errors.Is(renewErr, context.Canceled) || errors.Is(renewErr, context.DeadlineExceeded) {
				return result, renewErr
			}
			if isRenewNotYetAvailableError(renewErr) {
				continue
			}

			result.Failed++
			domain := target.DomainName()
			if domain == "" {
				domain = fmt.Sprintf("id-%d", target.ID)
			}
			result.FailedList = append(result.FailedList, report.FailedDomain{
				Domain: domain,
				Reason: renewErr.Error(),
			})

			reason := strings.TrimSpace(renewErr.Error())
			if reason == "" {
				reason = "request failed"
			}
			if _, ok := failureReasonSeen[reason]; !ok && len(failureReasons) < 3 {
				failureReasonSeen[reason] = struct{}{}
				failureReasons = append(failureReasons, reason)
			}
			continue
		}

		result.Renewed++
		successDomain := strings.TrimSpace(renewResult.Subdomain)
		if successDomain == "" {
			successDomain = target.DomainName()
		}
		result.RenewedList = append(result.RenewedList, report.RenewedDomain{
			Domain:        successDomain,
			NewExpiresAt:  renewResult.NewExpiresAt,
			RemainingDays: renewResult.RemainingDays,
		})
	}

	if result.Failed > 0 {
		return result, &renewFailuresError{
			Count:   result.Failed,
			Reasons: failureReasons,
		}
	}
	return result, nil
}

// listSubdomainsWithRateLimitRetry 对子域名列表请求中的 DNSHE 限流错误做有限退避重试。
func (s *Service) listSubdomainsWithRateLimitRetry(ctx context.Context) ([]dnshe.Subdomain, error) {
	return callWithRateLimitRetry(ctx, func() ([]dnshe.Subdomain, error) {
		return s.dnsClient.ListSubdomains(ctx)
	})
}

// renewSubdomainWithRateLimitRetry 对续期请求中的 DNSHE 限流错误做有限退避重试。
func (s *Service) renewSubdomainWithRateLimitRetry(ctx context.Context, subdomainID int) (dnshe.RenewResult, error) {
	return callWithRateLimitRetry(ctx, func() (dnshe.RenewResult, error) {
		return s.dnsClient.RenewSubdomain(ctx, subdomainID)
	})
}

// callWithRateLimitRetry 对单个 DNSHE API 调用套用限流退避策略。
func callWithRateLimitRetry[T any](ctx context.Context, call func() (T, error)) (T, error) {
	var result T
	for attempt := 0; ; attempt++ {
		callResult, err := call()
		if err == nil {
			return callResult, nil
		}
		result = callResult

		if attempt >= maxRateLimitRetries || !isRateLimitError(err) {
			return result, err
		}

		delay := rateLimitRetryInterval(time.Now().UTC(), err, attempt)
		if waitErr := sleepContext(ctx, delay); waitErr != nil {
			return result, waitErr
		}
	}
}

// isRateLimitError 判断错误是否为 DNSHE V2 限流错误。
func isRateLimitError(err error) bool {
	var apiErr *dnshe.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.ErrorCode == rateLimitErrorCode || apiErr.StatusCode == http.StatusTooManyRequests
}

// rateLimitRetryInterval 优先使用 ResetAt，缺失或不可解析时退回到短递增间隔。
func rateLimitRetryInterval(now time.Time, err error, attempt int) time.Duration {
	var apiErr *dnshe.APIError
	if errors.As(err, &apiErr) {
		if resetAt, ok := parseSubdomainTime(apiErr.ResetAt); ok && resetAt.After(now) {
			delay := resetAt.Sub(now)
			if delay > maxRateLimitRetryInterval {
				return maxRateLimitRetryInterval
			}
			return delay
		}
	}

	delay := time.Duration(attempt+1) * time.Second
	if delay > maxRateLimitRetryInterval {
		return maxRateLimitRetryInterval
	}
	return delay
}

// sleepContext 在等待退避间隔时响应整体超时和系统信号取消。
func sleepContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// describeDomains 汇总账号下子域名的可展示域名和到期时间。
func describeDomains(now time.Time, targets []dnshe.Subdomain) []report.DomainInfo {
	domains := make([]report.DomainInfo, 0, len(targets))
	for _, target := range targets {
		domain := target.DomainName()
		if domain == "" {
			domain = fmt.Sprintf("id-%d", target.ID)
		}
		domains = append(domains, report.DomainInfo{
			Domain:        domain,
			ExpiresAt:     resolveSubdomainExpiry(target),
			RemainingDays: resolveRemainingDays(now, target),
		})
	}
	return domains
}

// filterRenewableTargets 只保留剩余时间小于 180 天的子域名。
func filterRenewableTargets(now time.Time, targets []dnshe.Subdomain) []dnshe.Subdomain {
	eligible := make([]dnshe.Subdomain, 0, len(targets))
	for _, target := range targets {
		if shouldRenewSubdomain(now, target) {
			eligible = append(eligible, target)
		}
	}
	return eligible
}

// shouldRenewSubdomain 判断子域名是否进入续期窗口。
func shouldRenewSubdomain(now time.Time, target dnshe.Subdomain) bool {
	if target.NeverExpires() {
		return false
	}

	if target.RemainingDays != nil {
		return *target.RemainingDays < renewWindowDays
	}

	if expiresAt, ok := parseSubdomainTime(target.ExpiresAt); ok {
		return *daysUntil(now, expiresAt) < renewWindowDays
	}

	baseTime, ok := parseLatestSubdomainTimestamp(target)
	if !ok {
		return false
	}

	return *daysUntil(now, baseTime.AddDate(1, 0, 0)) < renewWindowDays
}

// resolveSubdomainExpiry 返回子域名的可展示到期时间。
func resolveSubdomainExpiry(target dnshe.Subdomain) string {
	if target.NeverExpires() {
		return "never"
	}
	if expiresAt, ok := parseSubdomainTime(target.ExpiresAt); ok {
		return expiresAt.Format("2006-01-02 15:04:05")
	}
	if baseTime, ok := parseLatestSubdomainTimestamp(target); ok {
		return baseTime.AddDate(1, 0, 0).Format("2006-01-02 15:04:05")
	}
	return strings.TrimSpace(target.ExpiresAt)
}

// resolveRemainingDays 返回子域名剩余天数；如果无法推断则返回 nil。
func resolveRemainingDays(now time.Time, target dnshe.Subdomain) *int {
	if target.NeverExpires() {
		return nil
	}
	if target.RemainingDays != nil {
		days := *target.RemainingDays
		return &days
	}
	if expiresAt, ok := parseSubdomainTime(target.ExpiresAt); ok {
		return daysUntil(now, expiresAt)
	}
	if baseTime, ok := parseLatestSubdomainTimestamp(target); ok {
		return daysUntil(now, baseTime.AddDate(1, 0, 0))
	}
	return nil
}

// daysUntil 以自然日近似返回从当前时间到目标时间的剩余天数。
func daysUntil(now time.Time, target time.Time) *int {
	days := dateOrdinal(target.UTC()) - dateOrdinal(now.UTC())
	return &days
}

// dateOrdinal 返回公历日期序号，避免远未来日期触发 time.Duration 上限。
func dateOrdinal(ts time.Time) int {
	year := ts.Year()
	previousYear := year - 1
	return previousYear*365 + previousYear/4 - previousYear/100 + previousYear/400 + ts.YearDay()
}

// parseLatestSubdomainTimestamp 取 created_at 和 updated_at 中较新的时间。
func parseLatestSubdomainTimestamp(target dnshe.Subdomain) (time.Time, bool) {
	updatedAt, updatedOK := parseSubdomainTime(target.UpdatedAt)
	createdAt, createdOK := parseSubdomainTime(target.CreatedAt)

	switch {
	case updatedOK && createdOK:
		if updatedAt.After(createdAt) {
			return updatedAt, true
		}
		return createdAt, true
	case updatedOK:
		return updatedAt, true
	case createdOK:
		return createdAt, true
	default:
		return time.Time{}, false
	}
}

// parseSubdomainTime 解析 DNSHE 返回的时间字符串。
func parseSubdomainTime(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}

	layouts := []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
		time.RFC3339Nano,
	}
	for _, layout := range layouts {
		if ts, err := time.ParseInLocation(layout, raw, time.UTC); err == nil {
			return ts.UTC(), true
		}
	}
	return time.Time{}, false
}

// isRenewNotYetAvailableError 判断接口是否明确表示尚未进入续期窗口。
func isRenewNotYetAvailableError(err error) bool {
	var apiErr *dnshe.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	if apiErr.StatusCode != 422 {
		return false
	}

	message := strings.ToLower(strings.TrimSpace(apiErr.ErrorText))
	if message == "" {
		message = strings.ToLower(strings.TrimSpace(apiErr.Message))
	}
	return strings.Contains(message, "renewal not yet available")
}

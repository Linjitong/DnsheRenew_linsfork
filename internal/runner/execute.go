package runner

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"dnsherene/internal/app"
	"dnsherene/internal/config"
	"dnsherene/internal/output"
	"dnsherene/internal/report"
	"dnsherene/pkg/dnshe"
)

// Execute 顺序执行全部 API 凭证的续期流程并返回结构化报告。
//
// 当存在多组凭证时，单组失败不会中断后续组执行，最终统一聚合错误返回。
func Execute(ctx context.Context, cfg config.Config) (report.Info, error) {
	info := report.Info{
		GeneratedAt: time.Now().UTC(),
		Accounts:    make([]report.AccountInfo, 0, len(cfg.Credentials)),
	}

	var runErrs []error
	total := len(cfg.Credentials)
	for i, cred := range cfg.Credentials {
		if err := ctx.Err(); err != nil {
			runErrs = append(runErrs, err)
			break
		}

		account, err := runAccount(ctx, cfg, cred, i+1, total)
		info.RenewedTotal += account.Renewed
		info.Accounts = append(info.Accounts, account)
		if err != nil {
			runErrs = append(runErrs, err)
			if ctx.Err() != nil {
				break
			}
		}
	}

	if len(runErrs) > 0 {
		return info, errors.Join(runErrs...)
	}
	return info, nil
}

// runAccount 执行单个 API 账号的一次完整续期任务。
func runAccount(
	ctx context.Context,
	cfg config.Config,
	cred config.APICredential,
	index int,
	total int,
) (report.AccountInfo, error) {
	account := report.AccountInfo{
		Index:        index,
		Total:        total,
		APIKeyMasked: output.MaskAPIKey(cred.APIKey),
	}

	dnsClient, err := dnshe.NewClient(dnshe.Config{
		BaseURL:   cfg.APIBaseURL,
		APIKey:    cred.APIKey,
		APISecret: cred.APISecret,
		HTTPClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	})
	if err != nil {
		account.Error = output.SanitizePublicError(err)
		return account, fmt.Errorf("init dnshe sdk for api[%d] failed: %w", index, err)
	}

	service, err := app.NewService(dnsClient)
	if err != nil {
		account.Error = output.SanitizePublicError(err)
		return account, fmt.Errorf("init service for api[%d] failed: %w", index, err)
	}

	result, err := service.Run(ctx, cfg.DryRun)
	account.Matched = result.Matched
	account.Renewed = result.Renewed
	account.Failed = result.Failed
	account.DryRun = result.DryRun
	account.Domains = result.Domains
	account.RenewedList = result.RenewedList
	account.FailedList = result.FailedList
	for i := range account.FailedList {
		account.FailedList[i].Reason = output.SanitizePublicText(account.FailedList[i].Reason)
	}
	if err != nil {
		account.Error = output.SanitizePublicError(err)
		return account, fmt.Errorf("run renew for api[%d] failed: %w", index, err)
	}

	return account, nil
}

package dnshe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type baseResponse struct {
	Success   bool            `json:"success"`
	Message   string          `json:"message"`
	Error     string          `json:"error"`
	ErrorCode string          `json:"error_code"`
	Details   apiErrorDetails `json:"details"`
}

func (r baseResponse) detailsMap() map[string]any {
	return r.Details.Map
}

type apiError struct {
	Error     string          `json:"error"`
	Message   string          `json:"message"`
	ErrorCode string          `json:"error_code"`
	Details   apiErrorDetails `json:"details"`
	Limit     *int            `json:"limit"`
	Remaining *int            `json:"remaining"`
	ResetAt   string          `json:"reset_at"`
}

type apiErrorDetails struct {
	RequestID string
	Limit     *int
	Remaining *int
	ResetAt   string
	Map       map[string]any
}

func (d *apiErrorDetails) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	d.Map = raw
	if value, ok := raw["request_id"].(string); ok {
		d.RequestID = value
	}
	if value, ok := numberFromAny(raw["limit"]); ok {
		d.Limit = &value
	}
	if value, ok := numberFromAny(raw["remaining"]); ok {
		d.Remaining = &value
	}
	if value, ok := raw["reset_at"].(string); ok {
		d.ResetAt = value
	}
	return nil
}

// requestJSON 是统一 HTTP 调用入口，负责公共 query/header 与通用错误处理。
func (c *Client) requestJSON(
	ctx context.Context,
	method string,
	endpoint string,
	action string,
	query url.Values,
	body any,
	out any,
) error {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return fmt.Errorf("invalid base URL: %w", err)
	}

	q := u.Query()
	q.Set("m", "domain_hub")
	q.Set("endpoint", endpoint)
	if action != "" {
		q.Set("action", action)
	}
	for key, values := range query {
		for _, value := range values {
			q.Add(key, value)
		}
	}
	u.RawQuery = q.Encode()

	var bodyReader io.Reader
	if body != nil {
		raw, marshalErr := json.Marshal(body)
		if marshalErr != nil {
			return fmt.Errorf("marshal request body failed: %w", marshalErr)
		}
		bodyReader = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return fmt.Errorf("build request failed: %w", err)
	}
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}
	if c.apiSecret != "" {
		req.Header.Set("X-API-Secret", c.apiSecret)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		return decodeAPIError(resp.StatusCode, respBody)
	}

	if out == nil {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("decode response failed: %w; body=%s", err, truncate(respBody, 512))
	}
	return nil
}

// ensureSuccess 校验业务响应中的 success 字段并转换为结构化错误。
func ensureSuccess(operation string, resp baseResponse) error {
	if resp.Success {
		return nil
	}
	return &APIError{
		Operation: operation,
		ErrorCode: strings.TrimSpace(resp.ErrorCode),
		ErrorText: strings.TrimSpace(resp.Error),
		Message:   strings.TrimSpace(resp.Message),
		RequestID: strings.TrimSpace(resp.Details.RequestID),
		Limit:     resp.Details.Limit,
		Remaining: resp.Details.Remaining,
		ResetAt:   strings.TrimSpace(resp.Details.ResetAt),
		Details:   resp.detailsMap(),
	}
}

// decodeAPIError 将 HTTP 错误响应体解析为 APIError。
func decodeAPIError(statusCode int, body []byte) error {
	errResp := &APIError{
		StatusCode: statusCode,
		RawBody:    truncate(body, 256),
	}

	var payload apiError
	if err := json.Unmarshal(body, &payload); err == nil {
		apiErr := &APIError{
			StatusCode: statusCode,
			ErrorCode:  strings.TrimSpace(payload.ErrorCode),
			ErrorText:  strings.TrimSpace(payload.Error),
			Message:    strings.TrimSpace(payload.Message),
			RequestID:  strings.TrimSpace(payload.Details.RequestID),
			Limit:      firstIntPtr(payload.Limit, payload.Details.Limit),
			Remaining:  firstIntPtr(payload.Remaining, payload.Details.Remaining),
			ResetAt:    firstNonEmpty(payload.ResetAt, payload.Details.ResetAt),
			Details:    payload.Details.Map,
			RawBody:    truncate(body, 256),
		}
		return apiErr
	}

	if len(body) == 0 {
		errResp.RawBody = "empty response"
	}
	return errResp
}

func numberFromAny(value any) (int, bool) {
	switch typed := value.(type) {
	case float64:
		return int(typed), true
	case int:
		return typed, true
	default:
		return 0, false
	}
}

func firstIntPtr(values ...*int) *int {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

// truncate 截断过长响应体，避免错误信息无限增长。
func truncate(in []byte, max int) string {
	s := strings.TrimSpace(string(in))
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// firstNonEmpty 返回第一个去空格后非空的字符串。
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

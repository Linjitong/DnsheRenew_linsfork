package dnshe

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type listAPIKeysResponse struct {
	baseResponse
	Count int      `json:"count"`
	Keys  []APIKey `json:"keys"`
}

type createAPIKeyResponse struct {
	baseResponse
	APIKey    string `json:"api_key"`
	APISecret string `json:"api_secret"`
	Warning   string `json:"warning"`
}

type regenerateAPIKeyResponse struct {
	baseResponse
	APIKey    string `json:"api_key"`
	APISecret string `json:"api_secret"`
	Warning   string `json:"warning"`
}

type deleteAPIKeyResponse struct {
	baseResponse
}

// ListAPIKeys 获取当前账号全部 API Key 列表。
//
// Deprecated: 请使用 ListAPIKeysWithResult，以便读取 count 元数据。
func (c *Client) ListAPIKeys(ctx context.Context) ([]APIKey, error) {
	result, err := c.ListAPIKeysWithResult(ctx)
	if err != nil {
		return nil, err
	}
	return result.Keys, nil
}

// ListAPIKeysWithResult 获取当前账号全部 API Key 列表，并返回服务端 count。
func (c *Client) ListAPIKeysWithResult(ctx context.Context) (ListAPIKeysResult, error) {
	var out listAPIKeysResponse
	if err := c.requestJSON(ctx, http.MethodGet, "keys", "list", nil, nil, &out); err != nil {
		return ListAPIKeysResult{}, err
	}
	if err := ensureSuccess("list api keys", out.baseResponse); err != nil {
		return ListAPIKeysResult{}, err
	}
	return ListAPIKeysResult{
		Count: out.Count,
		Keys:  out.Keys,
	}, nil
}

// CreateAPIKey 创建新的 API Key。
func (c *Client) CreateAPIKey(ctx context.Context, req CreateAPIKeyRequest) (CreateAPIKeyResult, error) {
	keyName := strings.TrimSpace(req.KeyName)
	if keyName == "" {
		return CreateAPIKeyResult{}, fmt.Errorf("keyName is required")
	}

	payload := map[string]any{
		"key_name": keyName,
	}
	if whitelist := strings.TrimSpace(req.IPWhitelist); whitelist != "" {
		payload["ip_whitelist"] = whitelist
	}

	var out createAPIKeyResponse
	if err := c.requestJSON(ctx, http.MethodPost, "keys", "create", nil, payload, &out); err != nil {
		return CreateAPIKeyResult{}, err
	}
	if err := ensureSuccess("create api key", out.baseResponse); err != nil {
		return CreateAPIKeyResult{}, err
	}

	return CreateAPIKeyResult{
		APIKey:    out.APIKey,
		APISecret: out.APISecret,
		Warning:   strings.TrimSpace(out.Warning),
		Message:   strings.TrimSpace(out.Message),
	}, nil
}

// DeleteAPIKey 删除指定 API Key。
//
// Deprecated: 请使用 DeleteAPIKeyWithResult，以便读取服务端返回消息。
func (c *Client) DeleteAPIKey(ctx context.Context, keyID int) error {
	_, err := c.DeleteAPIKeyWithResult(ctx, keyID)
	return err
}

// DeleteAPIKeyWithResult 删除指定 API Key，并返回服务端消息。
func (c *Client) DeleteAPIKeyWithResult(ctx context.Context, keyID int) (DeleteAPIKeyResult, error) {
	if keyID <= 0 {
		return DeleteAPIKeyResult{}, fmt.Errorf("keyID must be positive")
	}

	payload := map[string]any{
		"key_id": keyID,
	}

	var out deleteAPIKeyResponse
	if err := c.requestJSON(ctx, http.MethodPost, "keys", "delete", nil, payload, &out); err != nil {
		return DeleteAPIKeyResult{}, err
	}
	if err := ensureSuccess("delete api key", out.baseResponse); err != nil {
		return DeleteAPIKeyResult{}, err
	}
	return DeleteAPIKeyResult{Message: strings.TrimSpace(out.Message)}, nil
}

// RegenerateAPIKey 重置指定 API Key 的 secret。
func (c *Client) RegenerateAPIKey(ctx context.Context, keyID int) (RegenerateAPIKeyResult, error) {
	if keyID <= 0 {
		return RegenerateAPIKeyResult{}, fmt.Errorf("keyID must be positive")
	}

	payload := map[string]any{
		"key_id": keyID,
	}

	var out regenerateAPIKeyResponse
	if err := c.requestJSON(ctx, http.MethodPost, "keys", "regenerate", nil, payload, &out); err != nil {
		return RegenerateAPIKeyResult{}, err
	}
	if err := ensureSuccess("regenerate api key", out.baseResponse); err != nil {
		return RegenerateAPIKeyResult{}, err
	}

	return RegenerateAPIKeyResult{
		APIKey:    out.APIKey,
		APISecret: out.APISecret,
		Warning:   strings.TrimSpace(out.Warning),
		Message:   strings.TrimSpace(out.Message),
	}, nil
}

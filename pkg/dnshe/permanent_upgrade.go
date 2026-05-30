package dnshe

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type listPermanentUpgradesResponse struct {
	baseResponse
	State PermanentUpgradeState `json:"state"`
}

type permanentUpgradeMutationResponse struct {
	baseResponse
	RequestID  int    `json:"request_id"`
	AssistCode string `json:"assist_code"`
}

// ListPermanentUpgrades 查询当前账号永久升级中心状态。
func (c *Client) ListPermanentUpgrades(ctx context.Context, opts PermanentUpgradeListOptions) (PermanentUpgradeState, error) {
	query := url.Values{}
	if opts.Page < 0 {
		return PermanentUpgradeState{}, fmt.Errorf("page must be positive")
	}
	if opts.PerPage < 0 {
		return PermanentUpgradeState{}, fmt.Errorf("perPage must be positive")
	}
	if opts.Page > 0 {
		query.Set("page", strconv.Itoa(opts.Page))
	}
	if opts.PerPage > 0 {
		query.Set("per_page", strconv.Itoa(opts.PerPage))
	}

	var out listPermanentUpgradesResponse
	if err := c.requestJSON(ctx, http.MethodGet, "permanent_upgrade", "list", query, nil, &out); err != nil {
		return PermanentUpgradeState{}, err
	}
	if err := ensureSuccess("list permanent upgrades", out.baseResponse); err != nil {
		return PermanentUpgradeState{}, err
	}
	return out.State, nil
}

// CreatePermanentUpgrade 创建永久升级任务。
func (c *Client) CreatePermanentUpgrade(ctx context.Context, subdomainID int) (PermanentUpgradeResult, error) {
	if subdomainID <= 0 {
		return PermanentUpgradeResult{}, fmt.Errorf("subdomainID must be positive")
	}

	payload := map[string]any{
		"subdomain_id": subdomainID,
	}
	return c.permanentUpgradeMutation(ctx, "create", payload, "create permanent upgrade")
}

// AssistPermanentUpgrade 使用邀请码助力永久升级任务。
func (c *Client) AssistPermanentUpgrade(ctx context.Context, assistCode string) (PermanentUpgradeResult, error) {
	assistCode = strings.TrimSpace(assistCode)
	if assistCode == "" {
		return PermanentUpgradeResult{}, fmt.Errorf("assistCode is required")
	}

	payload := map[string]any{
		"assist_code": assistCode,
	}
	return c.permanentUpgradeMutation(ctx, "assist", payload, "assist permanent upgrade")
}

// CancelPermanentUpgrade 取消永久升级任务。
func (c *Client) CancelPermanentUpgrade(ctx context.Context, requestID int) (PermanentUpgradeResult, error) {
	if requestID <= 0 {
		return PermanentUpgradeResult{}, fmt.Errorf("requestID must be positive")
	}

	payload := map[string]any{
		"request_id": requestID,
	}
	return c.permanentUpgradeMutation(ctx, "cancel", payload, "cancel permanent upgrade")
}

func (c *Client) permanentUpgradeMutation(ctx context.Context, action string, payload map[string]any, operation string) (PermanentUpgradeResult, error) {
	var out permanentUpgradeMutationResponse
	if err := c.requestJSON(ctx, http.MethodPost, "permanent_upgrade", action, nil, payload, &out); err != nil {
		return PermanentUpgradeResult{}, err
	}
	if err := ensureSuccess(operation, out.baseResponse); err != nil {
		return PermanentUpgradeResult{}, err
	}
	return PermanentUpgradeResult{
		RequestID:  out.RequestID,
		AssistCode: strings.TrimSpace(out.AssistCode),
		Message:    strings.TrimSpace(out.Message),
	}, nil
}

package dnshe

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type listSubdomainsResponse struct {
	baseResponse
	Count      int         `json:"count"`
	Subdomains []Subdomain `json:"subdomains"`
	Pagination Pagination  `json:"pagination"`
}

type registerSubdomainResponse struct {
	baseResponse
	SubdomainID int    `json:"subdomain_id"`
	FullDomain  string `json:"full_domain"`
}

type deleteSubdomainResponse struct {
	baseResponse
	SubdomainID       int    `json:"subdomain_id"`
	FullDomain        string `json:"full_domain"`
	DNSRecordsDeleted int    `json:"dns_records_deleted"`
}

type getSubdomainResponse struct {
	baseResponse
	Subdomain  Subdomain   `json:"subdomain"`
	DNSRecords []DNSRecord `json:"dns_records"`
	DNSCount   int         `json:"dns_count"`
}

type renewSubdomainResponse struct {
	baseResponse
	SubdomainID       int     `json:"subdomain_id"`
	Subdomain         string  `json:"subdomain"`
	PreviousExpiresAt string  `json:"previous_expires_at"`
	NewExpiresAt      string  `json:"new_expires_at"`
	RenewedAt         string  `json:"renewed_at"`
	NeverExpires      int     `json:"never_expires"`
	Status            string  `json:"status"`
	RemainingDays     int     `json:"remaining_days"`
	ChargedAmount     float64 `json:"charged_amount"`
}

// ListSubdomains 获取当前账号可见的子域名列表。
//
// Deprecated: 请使用 ListSubdomainsWithOptions，以便读取 count 和 pagination 元数据。
func (c *Client) ListSubdomains(ctx context.Context) ([]Subdomain, error) {
	result, err := c.ListSubdomainsWithOptions(ctx, ListSubdomainsOptions{})
	if err != nil {
		return nil, err
	}
	return result.Subdomains, nil
}

// ListSubdomainsWithOptions 获取当前账号可见的子域名列表，并返回分页上下文。
func (c *Client) ListSubdomainsWithOptions(ctx context.Context, opts ListSubdomainsOptions) (ListSubdomainsResult, error) {
	query, err := listSubdomainsQuery(opts)
	if err != nil {
		return ListSubdomainsResult{}, err
	}

	var out listSubdomainsResponse
	if err := c.requestJSON(ctx, http.MethodGet, "subdomains", "list", query, nil, &out); err != nil {
		return ListSubdomainsResult{}, err
	}
	if err := ensureSuccess("list subdomains", out.baseResponse); err != nil {
		return ListSubdomainsResult{}, err
	}
	return ListSubdomainsResult{
		Count:      out.Count,
		Subdomains: out.Subdomains,
		Pagination: out.Pagination,
	}, nil
}

// RegisterSubdomain 注册新的子域名。
func (c *Client) RegisterSubdomain(ctx context.Context, req RegisterSubdomainRequest) (RegisterSubdomainResult, error) {
	subdomain := strings.TrimSpace(req.Subdomain)
	if subdomain == "" {
		return RegisterSubdomainResult{}, fmt.Errorf("subdomain is required")
	}
	rootdomain := strings.TrimSpace(req.Rootdomain)
	if rootdomain == "" {
		return RegisterSubdomainResult{}, fmt.Errorf("rootdomain is required")
	}

	payload := map[string]any{
		"subdomain":  subdomain,
		"rootdomain": rootdomain,
	}

	var out registerSubdomainResponse
	if err := c.requestJSON(ctx, http.MethodPost, "subdomains", "register", nil, payload, &out); err != nil {
		return RegisterSubdomainResult{}, err
	}
	if err := ensureSuccess("register subdomain", out.baseResponse); err != nil {
		return RegisterSubdomainResult{}, err
	}

	return RegisterSubdomainResult{
		SubdomainID: out.SubdomainID,
		FullDomain:  out.FullDomain,
		Message:     strings.TrimSpace(out.Message),
	}, nil
}

// GetSubdomain 获取单个子域名详情及其 DNS 记录。
func (c *Client) GetSubdomain(ctx context.Context, subdomainID int) (SubdomainDetail, error) {
	if subdomainID <= 0 {
		return SubdomainDetail{}, fmt.Errorf("subdomainID must be positive")
	}

	query := url.Values{}
	query.Set("subdomain_id", strconv.Itoa(subdomainID))

	var out getSubdomainResponse
	if err := c.requestJSON(ctx, http.MethodGet, "subdomains", "get", query, nil, &out); err != nil {
		return SubdomainDetail{}, err
	}
	if err := ensureSuccess("get subdomain", out.baseResponse); err != nil {
		return SubdomainDetail{}, err
	}

	return SubdomainDetail{
		Subdomain:  out.Subdomain,
		DNSRecords: out.DNSRecords,
		DNSCount:   out.DNSCount,
	}, nil
}

// DeleteSubdomain 删除指定子域名。
//
// Deprecated: 请使用 DeleteSubdomainWithResult，以便读取 full_domain 和 dns_records_deleted。
func (c *Client) DeleteSubdomain(ctx context.Context, subdomainID int) error {
	_, err := c.DeleteSubdomainWithResult(ctx, subdomainID)
	return err
}

// DeleteSubdomainWithResult 删除指定子域名，并返回服务端提供的删除结果。
func (c *Client) DeleteSubdomainWithResult(ctx context.Context, subdomainID int) (DeleteSubdomainResult, error) {
	if subdomainID <= 0 {
		return DeleteSubdomainResult{}, fmt.Errorf("subdomainID must be positive")
	}

	payload := map[string]any{
		"subdomain_id": subdomainID,
	}

	var out deleteSubdomainResponse
	if err := c.requestJSON(ctx, http.MethodPost, "subdomains", "delete", nil, payload, &out); err != nil {
		return DeleteSubdomainResult{}, err
	}
	if err := ensureSuccess("delete subdomain", out.baseResponse); err != nil {
		return DeleteSubdomainResult{}, err
	}
	return DeleteSubdomainResult{
		SubdomainID:       out.SubdomainID,
		FullDomain:        out.FullDomain,
		DNSRecordsDeleted: out.DNSRecordsDeleted,
		Message:           strings.TrimSpace(out.Message),
	}, nil
}

// RenewSubdomain 按子域名 ID 发起续期请求。
func (c *Client) RenewSubdomain(ctx context.Context, subdomainID int) (RenewResult, error) {
	if subdomainID <= 0 {
		return RenewResult{}, fmt.Errorf("subdomainID must be positive")
	}

	payload := map[string]any{
		"subdomain_id": subdomainID,
	}

	var out renewSubdomainResponse
	if err := c.requestJSON(ctx, http.MethodPost, "subdomains", "renew", nil, payload, &out); err != nil {
		return RenewResult{}, err
	}
	if err := ensureSuccess("renew subdomain", out.baseResponse); err != nil {
		return RenewResult{}, err
	}

	return RenewResult{
		SubdomainID:       out.SubdomainID,
		Message:           strings.TrimSpace(out.Message),
		Subdomain:         out.Subdomain,
		PreviousExpiresAt: out.PreviousExpiresAt,
		NewExpiresAt:      out.NewExpiresAt,
		RenewedAt:         out.RenewedAt,
		NeverExpires:      out.NeverExpires != 0,
		Status:            out.Status,
		RemainingDays:     out.RemainingDays,
		ChargedAmount:     out.ChargedAmount,
	}, nil
}

func listSubdomainsQuery(opts ListSubdomainsOptions) (url.Values, error) {
	query := url.Values{}
	if opts.Page < 0 {
		return nil, fmt.Errorf("page must be positive")
	}
	if opts.PerPage < 0 {
		return nil, fmt.Errorf("perPage must be positive")
	}
	if opts.Page > 0 {
		query.Set("page", strconv.Itoa(opts.Page))
	}
	if opts.PerPage > 0 {
		query.Set("per_page", strconv.Itoa(opts.PerPage))
	}
	if opts.IncludeTotal {
		query.Set("include_total", "1")
	}
	if search := strings.TrimSpace(opts.Search); search != "" {
		query.Set("search", search)
	}
	if rootdomain := strings.TrimSpace(opts.Rootdomain); rootdomain != "" {
		query.Set("rootdomain", rootdomain)
	}
	if status := strings.TrimSpace(opts.Status); status != "" {
		query.Set("status", status)
	}
	if createdFrom := strings.TrimSpace(opts.CreatedFrom); createdFrom != "" {
		query.Set("created_from", createdFrom)
	}
	if createdTo := strings.TrimSpace(opts.CreatedTo); createdTo != "" {
		query.Set("created_to", createdTo)
	}
	if sortBy := strings.TrimSpace(opts.SortBy); sortBy != "" {
		query.Set("sort_by", sortBy)
	}
	if sortDir := strings.TrimSpace(opts.SortDir); sortDir != "" {
		query.Set("sort_dir", sortDir)
	}
	if len(opts.Fields) > 0 {
		fields := make([]string, 0, len(opts.Fields))
		for _, field := range opts.Fields {
			if trimmed := strings.TrimSpace(field); trimmed != "" {
				fields = append(fields, trimmed)
			}
		}
		if len(fields) > 0 {
			query.Set("fields", strings.Join(fields, ","))
		}
	}
	return query, nil
}

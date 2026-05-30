package dnshe

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// PublicConfig 定义无需强制 API Key 的公共接口客户端初始化参数。
type PublicConfig struct {
	BaseURL    string       // API 基础地址；为空时使用 DefaultBaseURL。
	HTTPClient *http.Client // 自定义 HTTP 客户端；为空时使用 20 秒超时的默认客户端。
}

// NewPublicClient 创建可调用公共接口的客户端，例如 WHOIS 查询。
func NewPublicClient(cfg PublicConfig) *Client {
	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 20 * time.Second}
	}

	return &Client{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

type whoisResponse struct {
	baseResponse
	Domain          string     `json:"domain"`
	Registered      *bool      `json:"registered"`
	Status          string     `json:"status"`
	RegisteredAt    string     `json:"registered_at"`
	ExpiresAt       string     `json:"expires_at"`
	RegistrantEmail string     `json:"registrant_email"`
	Nameservers     []string   `json:"nameservers"`
	NameServers     []string   `json:"name_servers"`
	RateLimit       *RateLimit `json:"rate_limit"`
}

// Whois 查询内置子域名或外部域名的 WHOIS 信息。
func (c *Client) Whois(ctx context.Context, domain string) (WhoisResult, error) {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return WhoisResult{}, fmt.Errorf("domain is required")
	}

	query := url.Values{}
	query.Set("domain", domain)

	var out whoisResponse
	if err := c.requestJSON(ctx, http.MethodGet, "whois", "", query, nil, &out); err != nil {
		return WhoisResult{}, err
	}
	if err := ensureSuccess("whois", out.baseResponse); err != nil {
		return WhoisResult{}, err
	}

	registered := strings.TrimSpace(out.Status) != "unregistered"
	if out.Registered != nil {
		registered = *out.Registered
	}
	nameServers := out.NameServers
	if len(nameServers) == 0 {
		nameServers = out.Nameservers
	}
	nameservers := out.Nameservers
	if len(nameservers) == 0 {
		nameservers = out.NameServers
	}

	return WhoisResult{
		Domain:          out.Domain,
		Registered:      registered,
		Status:          out.Status,
		Message:         strings.TrimSpace(out.Message),
		RegisteredAt:    out.RegisteredAt,
		ExpiresAt:       out.ExpiresAt,
		RegistrantEmail: out.RegistrantEmail,
		Nameservers:     nameservers,
		NameServers:     nameServers,
		RateLimit:       out.RateLimit,
	}, nil
}

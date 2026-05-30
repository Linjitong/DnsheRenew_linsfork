package dnshe

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// DefaultBaseURL 是 DNSHE API 的默认基础地址。
const DefaultBaseURL = "https://api005.dnshe.com/index.php"

// Config 定义 DNSHE SDK 的初始化参数。
type Config struct {
	BaseURL    string       // API 基础地址；为空时使用 DefaultBaseURL。
	APIKey     string       // DNSHE API Key，必填。
	APISecret  string       // DNSHE API Secret，必填。
	HTTPClient *http.Client // 自定义 HTTP 客户端；为空时使用 20 秒超时的默认客户端。
}

// Client 封装 DNSHE API 调用细节（鉴权、请求构建、错误解析）。
type Client struct {
	baseURL    string
	apiKey     string
	apiSecret  string
	httpClient *http.Client
}

// NewClient 创建 SDK 客户端，并做基础参数校验与默认值兜底。
func NewClient(cfg Config) (*Client, error) {
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("APIKey is required")
	}

	apiSecret := strings.TrimSpace(cfg.APISecret)
	if apiSecret == "" {
		return nil, fmt.Errorf("APISecret is required")
	}

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
		apiKey:     apiKey,
		apiSecret:  apiSecret,
		httpClient: httpClient,
	}, nil
}

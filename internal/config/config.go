package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// DefaultRunTimeout 是 CLI 单次执行的默认整体超时时间。
const DefaultRunTimeout = 10 * time.Minute

// APICredential 表示一组 DNSHE API 凭证。
type APICredential struct {
	// APIKey 是 DNSHE 提供的访问 Key。
	APIKey string
	// APISecret 是与 APIKey 配对的密钥。
	APISecret string
}

// Config 表示程序运行所需的全部环境配置。
type Config struct {
	// Credentials 是要执行续期任务的 API 凭证集合。
	Credentials []APICredential

	// APIBaseURL 是 DNSHE API 基础地址，可为空（SDK 会使用默认值）。
	APIBaseURL string
	// DryRun 为 true 时只做演练，不执行续期请求。
	DryRun bool
	// RunTimeout 是一次 CLI 主流程允许运行的最长时间。
	RunTimeout time.Duration
	// Notification 是通知模块所需的全部配置。
	Notification NotificationConfig
}

// Load 从环境变量加载并校验配置。
func Load() (Config, error) {
	return loadWithLookup(os.Getenv)
}

// loadWithLookup 支持通过注入 lookup 函数实现可测试的配置加载。
func loadWithLookup(lookup func(string) string) (Config, error) {
	cfg := Config{
		APIBaseURL: strings.TrimSpace(lookup("DNSHE_API_BASE_URL")),
		DryRun:     parseBool(lookup("DNSHE_DRY_RUN")),
		RunTimeout: DefaultRunTimeout,
	}

	runTimeout, err := parseRunTimeout(lookup("DNSHE_RUN_TIMEOUT"))
	if err != nil {
		return cfg, err
	}
	cfg.RunTimeout = runTimeout

	creds, err := resolveCredentials(
		strings.TrimSpace(lookup("DNSHE_API_KEYS")),
		strings.TrimSpace(lookup("DNSHE_API_SECRETS")),
	)
	if err != nil {
		return cfg, err
	}
	cfg.Credentials = creds

	cfg.Notification, err = loadNotificationConfig(lookup)
	if err != nil {
		return cfg, err
	}

	return cfg, nil
}

// resolveCredentials 解析并校验统一凭证模式（DNSHE_API_KEYS/DNSHE_API_SECRETS）。
//
// 规则：
// 1. DNSHE_API_KEYS 和 DNSHE_API_SECRETS 都是必填。
// 2. key/secret 数量必须一致。
// 3. 单账号也使用该模式，仅填 1 组。
func resolveCredentials(
	multiKeysRaw string,
	multiSecretsRaw string,
) ([]APICredential, error) {
	keys := splitList(multiKeysRaw)
	secrets := splitList(multiSecretsRaw)

	if len(keys) == 0 {
		return nil, fmt.Errorf("DNSHE_API_KEYS is required")
	}
	if len(secrets) == 0 {
		return nil, fmt.Errorf("DNSHE_API_SECRETS is required")
	}
	if len(keys) != len(secrets) {
		return nil, fmt.Errorf("DNSHE_API_KEYS and DNSHE_API_SECRETS length mismatch: %d != %d", len(keys), len(secrets))
	}

	creds := make([]APICredential, 0, len(keys))
	for i := range keys {
		key := strings.TrimSpace(keys[i])
		secret := strings.TrimSpace(secrets[i])
		if key == "" {
			return nil, fmt.Errorf("DNSHE_API_KEYS item at index %d is empty", i)
		}
		if secret == "" {
			return nil, fmt.Errorf("DNSHE_API_SECRETS item at index %d is empty", i)
		}
		creds = append(creds, APICredential{
			APIKey:    key,
			APISecret: secret,
		})
	}
	return creds, nil
}

// splitList 将多值字符串按逗号、分号、换行和制表符拆分。
func splitList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\r' || r == '\t'
	})

	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if item := strings.TrimSpace(part); item != "" {
			out = append(out, item)
		}
	}
	return out
}

// parseBool 解析通用布尔字符串，无法识别时返回 false。
func parseBool(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// parseRunTimeout 解析 DNSHE_RUN_TIMEOUT；空值使用默认整体超时。
func parseRunTimeout(raw string) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return DefaultRunTimeout, nil
	}

	duration, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("DNSHE_RUN_TIMEOUT must be a duration like 10m or 30s: %w", err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("DNSHE_RUN_TIMEOUT must be positive")
	}
	return duration, nil
}

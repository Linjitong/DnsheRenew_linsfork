package dnshe

import (
	"strconv"
	"strings"
)

// Subdomain 表示 DNSHE 返回的子域名对象。
type Subdomain struct {
	ID                int    `json:"id"`                  // DNSHE 内部子域名 ID。
	Subdomain         string `json:"subdomain"`           // 子域名前缀。
	Rootdomain        string `json:"rootdomain"`          // 根域名。
	FullDomain        string `json:"full_domain"`         // 完整域名。
	Status            string `json:"status"`              // 域名状态，例如 active/suspended/expired。
	CreatedAt         string `json:"created_at"`          // 创建时间，使用服务端返回的原始字符串。
	UpdatedAt         string `json:"updated_at"`          // 更新时间，使用服务端返回的原始字符串。
	ExpiresAt         string `json:"expires_at"`          // 到期时间，使用服务端返回的原始字符串。
	RemainingDays     *int   `json:"remaining_days"`      // 剩余天数；服务端未返回时为 nil。
	NeverExpiresRaw   *int   `json:"never_expires"`       // 永不过期原始标记，推荐使用 NeverExpires 方法读取布尔值。
	CloudflareZoneID  string `json:"cloudflare_zone_id"`  // Cloudflare Zone ID；仅在服务端返回时有值。
	ProviderAccountID string `json:"provider_account_id"` // DNS 服务商账号 ID；仅在服务端返回时有值。
}

// DomainName 返回子域名的可展示完整域名。
func (s Subdomain) DomainName() string {
	if full := strings.TrimSpace(s.FullDomain); full != "" {
		return full
	}
	if sub := strings.TrimSpace(s.Subdomain); sub != "" {
		if root := strings.TrimSpace(s.Rootdomain); root != "" {
			return sub + "." + root
		}
		return sub
	}
	if root := strings.TrimSpace(s.Rootdomain); root != "" {
		return root
	}
	return ""
}

// NeverExpires 返回该子域名是否标记为永不过期。
func (s Subdomain) NeverExpires() bool {
	return s.NeverExpiresRaw != nil && *s.NeverExpiresRaw != 0
}

// SubdomainDetail 表示子域名详情以及其 DNS 记录集合。
//
// DNSCount 来自服务端返回的 dns_count 字段，通常与 DNSRecords 长度一致，
// 但调用方应以服务端字段为准处理分页或裁剪后的响应。
type SubdomainDetail struct {
	Subdomain  Subdomain
	DNSRecords []DNSRecord
	DNSCount   int
}

// Pagination 表示支持分页的列表接口返回的分页信息。
type Pagination struct {
	Page     int  `json:"page"`      // 当前页码。
	PerPage  int  `json:"per_page"`  // 每页数量。
	HasMore  bool `json:"has_more"`  // 是否还有下一页。
	NextPage int  `json:"next_page"` // 下一页页码；没有下一页时通常为 0。
	PrevPage int  `json:"prev_page"` // 上一页页码；没有上一页时通常为 0。
	Total    int  `json:"total"`     // 总数量；仅 include_total 生效或服务端返回时有值。
}

// ListSubdomainsOptions 定义子域名列表的分页、筛选和排序参数。
type ListSubdomainsOptions struct {
	Page         int      // 页码，从 1 开始；0 表示使用服务端默认值。
	PerPage      int      // 每页数量；0 表示使用服务端默认值。
	IncludeTotal bool     // 是否请求 total，总量统计可能更慢。
	Search       string   // 搜索关键字，匹配 subdomain 或 rootdomain。
	Rootdomain   string   // 按根域名过滤。
	Status       string   // 按状态过滤，例如 active/suspended/expired。
	CreatedFrom  string   // 创建开始日期，格式 YYYY-MM-DD。
	CreatedTo    string   // 创建结束日期，格式 YYYY-MM-DD。
	SortBy       string   // 排序字段，例如 id/created_at/updated_at/expires_at/subdomain。
	SortDir      string   // 排序方向，asc 或 desc。
	Fields       []string // 返回字段列表，SDK 会以逗号拼接为 fields 参数。
}

// ListSubdomainsResult 定义带分页信息的子域名列表结果。
type ListSubdomainsResult struct {
	Count      int         // 当前响应返回的数量。
	Subdomains []Subdomain // 子域名列表。
	Pagination Pagination  // 分页信息；服务端未返回时为零值。
}

// RegisterSubdomainRequest 定义注册子域名请求参数。
type RegisterSubdomainRequest struct {
	Subdomain  string // 子域名前缀。
	Rootdomain string // 根域名。
}

// RegisterSubdomainResult 定义注册子域名返回结果。
type RegisterSubdomainResult struct {
	SubdomainID int    // 新注册的子域名 ID。
	FullDomain  string // 新注册的完整域名。
	Message     string // 服务端返回消息。
}

// DeleteSubdomainResult 定义删除子域名后的完整返回结果。
type DeleteSubdomainResult struct {
	SubdomainID       int    // 被删除的子域名 ID。
	FullDomain        string // 被删除的完整域名。
	DNSRecordsDeleted int    // 一并删除的 DNS 记录数量。
	Message           string // 服务端返回消息。
}

// RenewResult 定义续期子域名返回结果。
type RenewResult struct {
	SubdomainID       int     // 续期的子域名 ID。
	Message           string  // 服务端返回消息。
	Subdomain         string  // 子域名前缀或服务端返回的域名展示值。
	PreviousExpiresAt string  // 续期前到期时间。
	NewExpiresAt      string  // 续期后到期时间。
	RenewedAt         string  // 续期发生时间。
	NeverExpires      bool    // 是否永不过期。
	Status            string  // 续期后的状态。
	RemainingDays     int     // 续期后的剩余天数。
	ChargedAmount     float64 // 本次续期扣费金额；免费续期通常为 0。
}

// DNSRecord 表示 DNS 记录对象。
type DNSRecord struct {
	ID        int     `json:"id"`         // DNSHE 内部记录 ID。
	RecordID  string  `json:"record_id"`  // 云 DNS 服务商记录 ID。
	Name      string  `json:"name"`       // 记录名称。
	Type      string  `json:"type"`       // 记录类型，例如 A/AAAA/CNAME/MX/TXT/NS/SRV/CAA。
	Content   string  `json:"content"`    // 记录值。
	TTL       int     `json:"ttl"`        // TTL。
	Priority  *int    `json:"priority"`   // MX/SRV 优先级；服务端未返回时为 nil。
	Line      *string `json:"line"`       // 解析线路；服务端未返回时为 nil。
	Proxied   bool    `json:"proxied"`    // 是否启用代理；仅服务商支持时有意义。
	Status    string  `json:"status"`     // 记录状态。
	CreatedAt string  `json:"created_at"` // 创建时间，使用服务端返回的原始字符串。
	UpdatedAt string  `json:"updated_at"` // 更新时间，使用服务端返回的原始字符串。
}

// ListDNSRecordsResult 定义 DNS 记录列表接口的完整返回结果。
type ListDNSRecordsResult struct {
	Count   int         // 当前响应返回的记录数量。
	Records []DNSRecord // DNS 记录列表。
}

// CreateDNSRecordRequest 定义创建 DNS 记录请求参数。
type CreateDNSRecordRequest struct {
	SubdomainID  int    // 子域名 ID。
	Type         string // 记录类型，例如 A/AAAA/CNAME/MX/TXT/NS/SRV/CAA。
	Content      string // 记录值；SRV/CAA 可通过结构化字段自动组装时允许为空。
	Name         string // 记录名称；空或 @ 表示当前子域名本身，服务端也支持完整域名。
	TTL          int    // TTL；0 表示使用服务端默认值。
	Priority     *int   // MX/SRV 优先级；nil 表示使用服务端默认值。
	Line         string // 解析线路；空表示不传。
	RecordWeight *int   // SRV 权重。
	// Deprecated: 请使用 RecordWeight。该别名仅因 DNSHE API 文档同时接受 weight 而保留。
	Weight     *int // SRV 权重别名。
	RecordPort *int // SRV 端口。
	// Deprecated: 请使用 RecordPort。该别名仅因 DNSHE API 文档同时接受 port 而保留。
	Port         *int   // SRV 端口别名。
	RecordTarget string // SRV 目标主机。
	// Deprecated: 请使用 RecordTarget。该别名仅因 DNSHE API 文档同时接受 target 而保留。
	Target   string // SRV 目标主机别名。
	CAAFlag  *int   // CAA flag；nil 表示不传。
	CAATag   string // CAA tag，例如 issue。
	CAAValue string // CAA value。
}

// CreateDNSRecordResult 定义创建 DNS 记录返回结果。
type CreateDNSRecordResult struct {
	ID               int    // DNSHE 内部记录 ID。
	RecordID         int    // 兼容旧 SDK 的内部记录 ID；新代码建议使用 ID。
	ProviderRecordID string // 云 DNS 服务商记录 ID。
	Message          string // 服务端返回消息。
}

// UpdateDNSRecordRequest 定义更新 DNS 记录请求参数。
type UpdateDNSRecordRequest struct {
	ID               int     // DNSHE 内部记录 ID，推荐优先使用。
	RecordID         int     // 兼容旧 SDK 的记录定位字段；会作为 record_id 参数发送。
	ProviderRecordID string  // 云 DNS 服务商记录 ID；非空时优先作为 record_id 参数发送。
	Type             *string // 新记录类型；nil 表示不更新。
	Name             *string // 新记录名称；nil 表示不更新。
	Content          *string // 新记录值；nil 表示不更新。
	TTL              *int    // 新 TTL；nil 表示不更新。
	Priority         *int    // 新 MX/SRV 优先级；nil 表示不更新。
	Line             *string // 新解析线路；nil 表示不更新。
	RecordWeight     *int    // 新 SRV 权重；nil 表示不更新。
	// Deprecated: 请使用 RecordWeight。该别名仅因 DNSHE API 文档同时接受 weight 而保留。
	Weight     *int // 新 SRV 权重别名。
	RecordPort *int // 新 SRV 端口；nil 表示不更新。
	// Deprecated: 请使用 RecordPort。该别名仅因 DNSHE API 文档同时接受 port 而保留。
	Port         *int    // 新 SRV 端口别名。
	RecordTarget *string // 新 SRV 目标主机；nil 表示不更新。
	// Deprecated: 请使用 RecordTarget。该别名仅因 DNSHE API 文档同时接受 target 而保留。
	Target   *string // 新 SRV 目标主机别名。
	CAAFlag  *int    // 新 CAA flag；nil 表示不更新。
	CAATag   *string // 新 CAA tag；nil 表示不更新。
	CAAValue *string // 新 CAA value；nil 表示不更新。
}

// DNSRecordMutationResult 定义 DNS 记录创建或更新后返回的记录定位信息。
type DNSRecordMutationResult struct {
	ID               int    // DNSHE 内部记录 ID。
	RecordID         int    // 兼容旧 SDK 的内部记录 ID；新代码建议使用 ID。
	ProviderRecordID string // 云 DNS 服务商记录 ID。
	Message          string // 服务端返回消息。
}

// DeleteDNSRecordResult 定义删除 DNS 记录后的完整返回结果。
type DeleteDNSRecordResult struct {
	Message string // 服务端返回消息。
}

// APIKey 表示 API Key 元数据对象。
type APIKey struct {
	ID           int    `json:"id"`            // API Key ID。
	KeyName      string `json:"key_name"`      // API Key 名称。
	APIKey       string `json:"api_key"`       // API Key 明文标识。
	Status       string `json:"status"`        // Key 状态。
	RequestCount int    `json:"request_count"` // 请求次数统计。
	LastUsedAt   string `json:"last_used_at"`  // 最后使用时间，使用服务端返回的原始字符串。
	CreatedAt    string `json:"created_at"`    // 创建时间，使用服务端返回的原始字符串。
}

// ListAPIKeysResult 定义 API Key 列表接口的完整返回结果。
type ListAPIKeysResult struct {
	Count int      // 当前响应返回的 Key 数量。
	Keys  []APIKey // API Key 列表。
}

// CreateAPIKeyRequest 定义创建 API Key 请求参数。
type CreateAPIKeyRequest struct {
	KeyName     string // API Key 名称。
	IPWhitelist string // IP 白名单，支持逗号、换行或分号分隔；空表示不传。
}

// CreateAPIKeyResult 定义创建 API Key 返回结果。
type CreateAPIKeyResult struct {
	APIKey    string // 新创建的 API Key。
	APISecret string // 新创建的 API Secret；通常仅显示一次。
	Warning   string // 服务端安全提示。
	Message   string // 服务端返回消息。
}

// RegenerateAPIKeyResult 定义重置 API Key 返回结果。
type RegenerateAPIKeyResult struct {
	APIKey    string // 被重置的 API Key。
	APISecret string // 新生成的 API Secret；通常仅显示一次。
	Warning   string // 服务端安全提示。
	Message   string // 服务端返回消息。
}

// DeleteAPIKeyResult 定义删除 API Key 后的完整返回结果。
type DeleteAPIKeyResult struct {
	Message string // 服务端返回消息。
}

// Quota 表示免费域名额度信息。
type Quota struct {
	Used        int `json:"used"`         // 已使用额度。
	Base        int `json:"base"`         // 基础额度。
	InviteBonus int `json:"invite_bonus"` // 邀请奖励额度。
	Total       int `json:"total"`        // 总额度。
	Available   int `json:"available"`    // 可用额度。
}

// RateLimit 表示 WHOIS 等接口返回的限流上下文。
type RateLimit struct {
	Limit     int    `json:"limit"`     // 限流窗口内最大请求数。
	Remaining int    `json:"remaining"` // 当前窗口剩余请求数。
	ResetAt   string `json:"reset_at"`  // 限流重置时间，使用服务端返回的原始字符串。
}

// WhoisResult 表示 WHOIS 查询结果。
type WhoisResult struct {
	Domain          string     `json:"domain"`           // 查询的完整域名。
	Registered      bool       `json:"registered"`       // 是否已注册；服务端未返回 registered 时根据 status 推断。
	Status          string     `json:"status"`           // 域名状态。
	Message         string     `json:"message"`          // 服务端返回消息。
	RegisteredAt    string     `json:"registered_at"`    // 注册时间，使用服务端返回的原始字符串。
	ExpiresAt       string     `json:"expires_at"`       // 到期时间，使用服务端返回的原始字符串。
	RegistrantEmail string     `json:"registrant_email"` // 注册人邮箱；可能受隐私保护影响。
	Nameservers     []string   `json:"nameservers"`      // nameservers 字段；缺失时会从 name_servers 兼容填充。
	NameServers     []string   `json:"name_servers"`     // name_servers 兼容字段；缺失时会从 nameservers 填充。
	RateLimit       *RateLimit `json:"rate_limit"`       // 公共 WHOIS 模式下的限流信息；服务端未返回时为 nil。
}

// PermanentUpgradeListOptions 定义永久升级中心列表查询参数。
type PermanentUpgradeListOptions struct {
	Page    int // 页码，从 1 开始；0 表示使用服务端默认值。
	PerPage int // 每页数量；0 表示使用服务端默认值。
}

// PermanentUpgradeState 表示永久升级中心状态。
type PermanentUpgradeState struct {
	Requests           []PermanentUpgradeRequest `json:"requests"`            // 当前账号的永久升级任务。
	AssistLogs         []PermanentUpgradeAssist  `json:"assist_logs"`         // 助力记录。
	EligibleSubdomains []Subdomain               `json:"eligible_subdomains"` // 可创建永久升级任务的子域名。
}

// PermanentUpgradeRequest 表示永久升级任务。
type PermanentUpgradeRequest struct {
	RequestID   int    `json:"request_id"`   // 永久升级任务 ID。
	SubdomainID int    `json:"subdomain_id"` // 目标子域名 ID。
	Status      string `json:"status"`       // 任务状态。
	AssistCode  string `json:"assist_code"`  // 可分享的助力码。
	CreatedAt   string `json:"created_at"`   // 创建时间，使用服务端返回的原始字符串。
	UpdatedAt   string `json:"updated_at"`   // 更新时间，使用服务端返回的原始字符串。
}

// PermanentUpgradeAssist 表示永久升级助力记录。
type PermanentUpgradeAssist struct {
	AssistCode string `json:"assist_code"` // 助力码。
	CreatedAt  string `json:"created_at"`  // 助力时间，使用服务端返回的原始字符串。
}

// PermanentUpgradeResult 表示永久升级中心变更操作结果。
type PermanentUpgradeResult struct {
	RequestID  int    // 永久升级任务 ID。
	AssistCode string // 助力码；仅服务端返回时有值。
	Message    string // 服务端返回消息。
}

// APIError 表示 DNSHE API 返回的结构化错误。
//
// 当服务返回非 2xx HTTP 状态码，或者返回 success=false 的业务错误时，
// SDK 会优先返回该类型，便于调用方通过 errors.As 读取限流等上下文字段。
type APIError struct {
	Operation string // SDK 内部操作名称，便于定位失败接口。

	StatusCode int            // HTTP 状态码；业务错误但 HTTP 2xx 时为 0。
	ErrorCode  string         // V2 稳定错误码，例如 rate_limit_exceeded。
	ErrorText  string         // error 字段，兼容旧响应。
	Message    string         // message 字段。
	RequestID  string         // details.request_id，服务端未返回时为空。
	Limit      *int           // details.limit 或顶层 limit；未返回时为 nil。
	Remaining  *int           // details.remaining 或顶层 remaining；未返回时为 nil。
	ResetAt    string         // details.reset_at 或顶层 reset_at。
	Details    map[string]any // 服务端 details 原始对象，便于读取未来新增字段。
	RawBody    string         // 无法结构化解析时保留的截断响应体。
}

// Error 返回适合日志和上层处理的错误字符串。
func (e *APIError) Error() string {
	if e == nil {
		return "unknown error"
	}
	msg := firstNonEmpty(e.ErrorText, e.Message, e.ErrorCode, e.RawBody, "unknown error")

	if e.Operation != "" {
		return e.Operation + " failed: " + msg
	}
	if e.StatusCode > 0 {
		return "http " + strconv.Itoa(e.StatusCode) + ": " + msg
	}
	return msg
}

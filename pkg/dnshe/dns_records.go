package dnshe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

var supportedDNSRecordTypes = map[string]struct{}{
	"A":     {},
	"AAAA":  {},
	"CNAME": {},
	"MX":    {},
	"NS":    {},
	"SRV":   {},
	"TXT":   {},
	"CAA":   {},
}

type listDNSRecordsResponse struct {
	baseResponse
	Count   int         `json:"count"`
	Records []DNSRecord `json:"records"`
}

type createDNSRecordResponse struct {
	baseResponse
	ID       int           `json:"id"`
	RecordID recordIDValue `json:"record_id"`
}

type updateDNSRecordResponse struct {
	baseResponse
	ID       int           `json:"id"`
	RecordID recordIDValue `json:"record_id"`
}

type deleteDNSRecordResponse struct {
	baseResponse
}

// ListDNSRecords 获取指定子域名下的 DNS 记录列表。
//
// Deprecated: 请使用 ListDNSRecordsWithResult，以便读取 count 元数据。
func (c *Client) ListDNSRecords(ctx context.Context, subdomainID int) ([]DNSRecord, error) {
	result, err := c.ListDNSRecordsWithResult(ctx, subdomainID)
	if err != nil {
		return nil, err
	}
	return result.Records, nil
}

// ListDNSRecordsWithResult 获取指定子域名下的 DNS 记录列表，并返回服务端 count。
func (c *Client) ListDNSRecordsWithResult(ctx context.Context, subdomainID int) (ListDNSRecordsResult, error) {
	if subdomainID <= 0 {
		return ListDNSRecordsResult{}, fmt.Errorf("subdomainID must be positive")
	}

	query := url.Values{}
	query.Set("subdomain_id", strconv.Itoa(subdomainID))

	var out listDNSRecordsResponse
	if err := c.requestJSON(ctx, http.MethodGet, "dns_records", "list", query, nil, &out); err != nil {
		return ListDNSRecordsResult{}, err
	}
	if err := ensureSuccess("list dns records", out.baseResponse); err != nil {
		return ListDNSRecordsResult{}, err
	}
	return ListDNSRecordsResult{
		Count:   out.Count,
		Records: out.Records,
	}, nil
}

// CreateDNSRecord 为子域名创建 DNS 记录。
//
// SRV/CAA 记录既可以直接传 Content，也可以使用 RecordWeight、RecordPort、
// RecordTarget、CAAFlag、CAATag、CAAValue 等结构化字段。Weight、Port、Target
// 仅作为 API 文档别名保留，新代码应使用 Record* 字段。
func (c *Client) CreateDNSRecord(ctx context.Context, req CreateDNSRecordRequest) (CreateDNSRecordResult, error) {
	if req.SubdomainID <= 0 {
		return CreateDNSRecordResult{}, fmt.Errorf("subdomainID must be positive")
	}

	recordType, err := normalizeRecordType(req.Type)
	if err != nil {
		return CreateDNSRecordResult{}, err
	}

	content := strings.TrimSpace(req.Content)
	if content == "" && !canUseStructuredRecordContent(req) {
		return CreateDNSRecordResult{}, fmt.Errorf("content is required")
	}
	if req.TTL < 0 {
		return CreateDNSRecordResult{}, fmt.Errorf("ttl must be positive")
	}
	if req.Priority != nil && *req.Priority < 0 {
		return CreateDNSRecordResult{}, fmt.Errorf("priority must be zero or greater")
	}
	recordWeight, err := chooseAliasInt("recordWeight", req.RecordWeight, "weight", req.Weight)
	if err != nil {
		return CreateDNSRecordResult{}, err
	}
	if err := validateRecordNumericPtr("recordWeight", recordWeight); err != nil {
		return CreateDNSRecordResult{}, err
	}
	recordPort, err := chooseAliasInt("recordPort", req.RecordPort, "port", req.Port)
	if err != nil {
		return CreateDNSRecordResult{}, err
	}
	if err := validateRecordPort(recordPort); err != nil {
		return CreateDNSRecordResult{}, err
	}
	recordTarget, err := chooseAliasString("recordTarget", req.RecordTarget, "target", req.Target)
	if err != nil {
		return CreateDNSRecordResult{}, err
	}
	if err := validateRecordNumericPtr("caaFlag", req.CAAFlag); err != nil {
		return CreateDNSRecordResult{}, err
	}

	payload := map[string]any{
		"subdomain_id": req.SubdomainID,
		"type":         recordType,
	}
	if content != "" {
		payload["content"] = content
	}
	if name := strings.TrimSpace(req.Name); name != "" {
		payload["name"] = name
	}
	if req.TTL > 0 {
		payload["ttl"] = req.TTL
	}
	if req.Priority != nil {
		payload["priority"] = *req.Priority
	}
	if line := strings.TrimSpace(req.Line); line != "" {
		payload["line"] = line
	}
	addOptionalInt(payload, "record_weight", recordWeight)
	addOptionalInt(payload, "record_port", recordPort)
	if recordTarget != "" {
		payload["record_target"] = recordTarget
	}
	addOptionalInt(payload, "caa_flag", req.CAAFlag)
	if tag := strings.TrimSpace(req.CAATag); tag != "" {
		payload["caa_tag"] = tag
	}
	if value := strings.TrimSpace(req.CAAValue); value != "" {
		payload["caa_value"] = value
	}

	var out createDNSRecordResponse
	if err := c.requestJSON(ctx, http.MethodPost, "dns_records", "create", nil, payload, &out); err != nil {
		return CreateDNSRecordResult{}, err
	}
	if err := ensureSuccess("create dns record", out.baseResponse); err != nil {
		return CreateDNSRecordResult{}, err
	}

	return CreateDNSRecordResult{
		ID:               out.ID,
		RecordID:         out.resultID(),
		ProviderRecordID: out.RecordID.ProviderID,
		Message:          strings.TrimSpace(out.Message),
	}, nil
}

// UpdateDNSRecord 更新 DNS 记录字段。
//
// Deprecated: 请使用 UpdateDNSRecordWithResult，以便读取 API 返回的 id 和云服务商 record_id。
func (c *Client) UpdateDNSRecord(ctx context.Context, req UpdateDNSRecordRequest) error {
	_, err := c.UpdateDNSRecordWithResult(ctx, req)
	return err
}

// UpdateDNSRecordWithResult 更新 DNS 记录字段，并返回服务端记录定位信息。
func (c *Client) UpdateDNSRecordWithResult(ctx context.Context, req UpdateDNSRecordRequest) (DNSRecordMutationResult, error) {
	payload := map[string]any{}
	if req.ID > 0 {
		payload["id"] = req.ID
	}
	if providerRecordID := strings.TrimSpace(req.ProviderRecordID); providerRecordID != "" {
		payload["record_id"] = providerRecordID
	} else if req.RecordID > 0 {
		payload["record_id"] = req.RecordID
	}
	if len(payload) == 0 {
		return DNSRecordMutationResult{}, fmt.Errorf("recordID or id must be provided")
	}

	if req.Type != nil {
		recordType, err := normalizeRecordType(*req.Type)
		if err != nil {
			return DNSRecordMutationResult{}, err
		}
		payload["type"] = recordType
	}
	if req.Name != nil {
		payload["name"] = strings.TrimSpace(*req.Name)
	}
	if req.Content != nil {
		content := strings.TrimSpace(*req.Content)
		if content == "" {
			return DNSRecordMutationResult{}, fmt.Errorf("content must not be empty")
		}
		payload["content"] = content
	}
	if req.TTL != nil {
		if *req.TTL <= 0 {
			return DNSRecordMutationResult{}, fmt.Errorf("ttl must be positive")
		}
		payload["ttl"] = *req.TTL
	}
	if req.Priority != nil {
		if *req.Priority < 0 {
			return DNSRecordMutationResult{}, fmt.Errorf("priority must be zero or greater")
		}
		payload["priority"] = *req.Priority
	}
	if req.Line != nil {
		payload["line"] = strings.TrimSpace(*req.Line)
	}
	recordWeight, err := chooseAliasInt("recordWeight", req.RecordWeight, "weight", req.Weight)
	if err != nil {
		return DNSRecordMutationResult{}, err
	}
	if err := addOptionalUpdateInt(payload, "record_weight", recordWeight, "recordWeight"); err != nil {
		return DNSRecordMutationResult{}, err
	}
	recordPort, err := chooseAliasInt("recordPort", req.RecordPort, "port", req.Port)
	if err != nil {
		return DNSRecordMutationResult{}, err
	}
	if err := addOptionalUpdatePort(payload, recordPort); err != nil {
		return DNSRecordMutationResult{}, err
	}
	recordTarget, targetSet, err := chooseAliasStringPtr("recordTarget", req.RecordTarget, "target", req.Target)
	if err != nil {
		return DNSRecordMutationResult{}, err
	}
	if targetSet {
		payload["record_target"] = recordTarget
	}
	if err := addOptionalUpdateInt(payload, "caa_flag", req.CAAFlag, "caaFlag"); err != nil {
		return DNSRecordMutationResult{}, err
	}
	if req.CAATag != nil {
		trimmed := strings.TrimSpace(*req.CAATag)
		if trimmed == "" {
			return DNSRecordMutationResult{}, fmt.Errorf("caaTag must not be empty")
		}
		payload["caa_tag"] = trimmed
	}
	if req.CAAValue != nil {
		trimmed := strings.TrimSpace(*req.CAAValue)
		if trimmed == "" {
			return DNSRecordMutationResult{}, fmt.Errorf("caaValue must not be empty")
		}
		payload["caa_value"] = trimmed
	}
	if !hasRecordMutationField(payload) {
		return DNSRecordMutationResult{}, fmt.Errorf("at least one field must be set for update")
	}

	var out updateDNSRecordResponse
	if err := c.requestJSON(ctx, http.MethodPost, "dns_records", "update", nil, payload, &out); err != nil {
		return DNSRecordMutationResult{}, err
	}
	if err := ensureSuccess("update dns record", out.baseResponse); err != nil {
		return DNSRecordMutationResult{}, err
	}
	return DNSRecordMutationResult{
		ID:               out.ID,
		RecordID:         out.resultID(),
		ProviderRecordID: out.RecordID.ProviderID,
		Message:          strings.TrimSpace(out.Message),
	}, nil
}

// normalizeRecordType 规范化并校验 DNS 记录类型。
func normalizeRecordType(raw string) (string, error) {
	recordType := strings.ToUpper(strings.TrimSpace(raw))
	if recordType == "" {
		return "", fmt.Errorf("type is required")
	}
	if _, ok := supportedDNSRecordTypes[recordType]; !ok {
		return "", fmt.Errorf("unsupported dns record type: %s", recordType)
	}
	return recordType, nil
}

// DeleteDNSRecord 删除指定 DNS 记录。
//
// Deprecated: 请使用 DeleteDNSRecordByID 或 DeleteDNSRecordByProviderRecordID。
func (c *Client) DeleteDNSRecord(ctx context.Context, recordID int) error {
	_, err := c.DeleteDNSRecordByID(ctx, recordID)
	return err
}

// DeleteDNSRecordByID 按 DNSHE 内部记录 ID 删除 DNS 记录。
func (c *Client) DeleteDNSRecordByID(ctx context.Context, recordID int) (DeleteDNSRecordResult, error) {
	if recordID <= 0 {
		return DeleteDNSRecordResult{}, fmt.Errorf("recordID must be positive")
	}

	payload := map[string]any{
		"id": recordID,
	}

	var out deleteDNSRecordResponse
	if err := c.requestJSON(ctx, http.MethodPost, "dns_records", "delete", nil, payload, &out); err != nil {
		return DeleteDNSRecordResult{}, err
	}
	if err := ensureSuccess("delete dns record", out.baseResponse); err != nil {
		return DeleteDNSRecordResult{}, err
	}
	return DeleteDNSRecordResult{Message: strings.TrimSpace(out.Message)}, nil
}

// DeleteDNSRecordByProviderRecordID 删除指定云 DNS 服务商记录 ID 的 DNS 记录。
//
// Deprecated: 请使用 DeleteDNSRecordByProviderRecordIDWithResult，以便读取服务端返回消息。
func (c *Client) DeleteDNSRecordByProviderRecordID(ctx context.Context, recordID string) error {
	_, err := c.DeleteDNSRecordByProviderRecordIDWithResult(ctx, recordID)
	return err
}

// DeleteDNSRecordByProviderRecordIDWithResult 按云 DNS 服务商记录 ID 删除 DNS 记录。
func (c *Client) DeleteDNSRecordByProviderRecordIDWithResult(ctx context.Context, recordID string) (DeleteDNSRecordResult, error) {
	recordID = strings.TrimSpace(recordID)
	if recordID == "" {
		return DeleteDNSRecordResult{}, fmt.Errorf("recordID is required")
	}

	payload := map[string]any{
		"record_id": recordID,
	}

	var out deleteDNSRecordResponse
	if err := c.requestJSON(ctx, http.MethodPost, "dns_records", "delete", nil, payload, &out); err != nil {
		return DeleteDNSRecordResult{}, err
	}
	if err := ensureSuccess("delete dns record", out.baseResponse); err != nil {
		return DeleteDNSRecordResult{}, err
	}
	return DeleteDNSRecordResult{Message: strings.TrimSpace(out.Message)}, nil
}

type recordIDValue struct {
	InternalID int
	ProviderID string
}

func (v *recordIDValue) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}

	var intID int
	if err := json.Unmarshal(data, &intID); err == nil {
		v.InternalID = intID
		v.ProviderID = strconv.Itoa(intID)
		return nil
	}

	var stringID string
	if err := json.Unmarshal(data, &stringID); err != nil {
		return err
	}
	v.ProviderID = strings.TrimSpace(stringID)
	if parsed, err := strconv.Atoi(v.ProviderID); err == nil {
		v.InternalID = parsed
	}
	return nil
}

func (r createDNSRecordResponse) resultID() int {
	if r.ID > 0 {
		return r.ID
	}
	return r.RecordID.InternalID
}

func (r updateDNSRecordResponse) resultID() int {
	if r.ID > 0 {
		return r.ID
	}
	return r.RecordID.InternalID
}

func canUseStructuredRecordContent(req CreateDNSRecordRequest) bool {
	recordType := strings.ToUpper(strings.TrimSpace(req.Type))
	recordPort, portErr := chooseAliasInt("recordPort", req.RecordPort, "port", req.Port)
	recordTarget, targetErr := chooseAliasString("recordTarget", req.RecordTarget, "target", req.Target)
	if recordType == "SRV" {
		return portErr == nil && targetErr == nil && recordPort != nil && recordTarget != ""
	}
	if recordType == "CAA" {
		return strings.TrimSpace(req.CAAValue) != ""
	}
	return false
}

func validateRecordNumericPtr(name string, value *int) error {
	if value != nil && *value < 0 {
		return fmt.Errorf("%s must be zero or greater", name)
	}
	return nil
}

func validateRecordPort(value *int) error {
	if value != nil && (*value < 1 || *value > 65535) {
		return fmt.Errorf("recordPort must be between 1 and 65535")
	}
	return nil
}

func addOptionalInt(payload map[string]any, key string, value *int) {
	if value != nil {
		payload[key] = *value
	}
}

func addOptionalUpdateInt(payload map[string]any, key string, value *int, name string) error {
	if err := validateRecordNumericPtr(name, value); err != nil {
		return err
	}
	addOptionalInt(payload, key, value)
	return nil
}

func addOptionalUpdatePort(payload map[string]any, value *int) error {
	if err := validateRecordPort(value); err != nil {
		return err
	}
	addOptionalInt(payload, "record_port", value)
	return nil
}

func firstStringPtr(values ...*string) *string {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func chooseAliasInt(primaryName string, primary *int, aliasName string, alias *int) (*int, error) {
	if primary != nil && alias != nil && *primary != *alias {
		return nil, fmt.Errorf("%s and %s must not both be set with different values", primaryName, aliasName)
	}
	return firstIntPtr(primary, alias), nil
}

func chooseAliasString(primaryName string, primary string, aliasName string, alias string) (string, error) {
	primary = strings.TrimSpace(primary)
	alias = strings.TrimSpace(alias)
	if primary != "" && alias != "" && primary != alias {
		return "", fmt.Errorf("%s and %s must not both be set with different values", primaryName, aliasName)
	}
	return firstNonEmpty(primary, alias), nil
}

func chooseAliasStringPtr(primaryName string, primary *string, aliasName string, alias *string) (string, bool, error) {
	if primary == nil && alias == nil {
		return "", false, nil
	}

	primaryValue := ""
	if primary != nil {
		primaryValue = strings.TrimSpace(*primary)
	}
	aliasValue := ""
	if alias != nil {
		aliasValue = strings.TrimSpace(*alias)
	}
	if primaryValue != "" && aliasValue != "" && primaryValue != aliasValue {
		return "", false, fmt.Errorf("%s and %s must not both be set with different values", primaryName, aliasName)
	}

	value := firstNonEmpty(primaryValue, aliasValue)
	if value == "" {
		return "", false, fmt.Errorf("%s must not be empty", primaryName)
	}
	return value, true, nil
}

func hasRecordMutationField(payload map[string]any) bool {
	for key := range payload {
		if key != "id" && key != "record_id" {
			return true
		}
	}
	return false
}

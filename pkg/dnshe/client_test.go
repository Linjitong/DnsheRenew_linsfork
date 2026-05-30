package dnshe

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := NewClient(Config{
		BaseURL:    server.URL,
		APIKey:     "test-key",
		APISecret:  "test-secret",
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	return client
}

func TestListSubdomainsBuildsRequest(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if got := r.URL.Query().Get("m"); got != "domain_hub" {
			t.Fatalf("m = %q, want domain_hub", got)
		}
		if got := r.URL.Query().Get("endpoint"); got != "subdomains" {
			t.Fatalf("endpoint = %q, want subdomains", got)
		}
		if got := r.URL.Query().Get("action"); got != "list" {
			t.Fatalf("action = %q, want list", got)
		}
		if got := r.Header.Get("X-API-Key"); got != "test-key" {
			t.Fatalf("X-API-Key = %q, want test-key", got)
		}
		if got := r.Header.Get("X-API-Secret"); got != "test-secret" {
			t.Fatalf("X-API-Secret = %q, want test-secret", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"count":   1,
			"subdomains": []map[string]any{
				{
					"id":          1,
					"subdomain":   "api",
					"rootdomain":  "example.com",
					"full_domain": "api.example.com",
					"status":      "active",
				},
			},
		})
	})

	got, err := client.ListSubdomains(context.Background())
	if err != nil {
		t.Fatalf("ListSubdomains returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0].FullDomain != "api.example.com" {
		t.Fatalf("FullDomain = %q, want api.example.com", got[0].FullDomain)
	}
}

func TestListSubdomainsWithOptionsBuildsQueryAndReturnsPagination(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		query := r.URL.Query()
		wantQuery := map[string]string{
			"m":             "domain_hub",
			"endpoint":      "subdomains",
			"action":        "list",
			"page":          "2",
			"per_page":      "50",
			"include_total": "1",
			"search":        "test",
			"rootdomain":    "example.com",
			"status":        "active",
			"created_from":  "2025-01-01",
			"created_to":    "2025-01-31",
			"sort_by":       "created_at",
			"sort_dir":      "desc",
			"fields":        "id,subdomain,status",
		}
		for key, want := range wantQuery {
			if got := query.Get(key); got != want {
				t.Fatalf("%s = %q, want %q", key, got, want)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"count":   1,
			"subdomains": []map[string]any{
				{
					"id":        201,
					"subdomain": "app201",
					"status":    "active",
				},
			},
			"pagination": map[string]any{
				"page":      2,
				"per_page":  50,
				"has_more":  true,
				"next_page": 3,
				"prev_page": 1,
				"total":     12500,
			},
		})
	})

	got, err := client.ListSubdomainsWithOptions(context.Background(), ListSubdomainsOptions{
		Page:         2,
		PerPage:      50,
		IncludeTotal: true,
		Search:       "test",
		Rootdomain:   "example.com",
		Status:       "active",
		CreatedFrom:  "2025-01-01",
		CreatedTo:    "2025-01-31",
		SortBy:       "created_at",
		SortDir:      "desc",
		Fields:       []string{"id", "subdomain", "status"},
	})
	if err != nil {
		t.Fatalf("ListSubdomainsWithOptions returned error: %v", err)
	}
	if got.Count != 1 {
		t.Fatalf("Count = %d, want 1", got.Count)
	}
	if len(got.Subdomains) != 1 || got.Subdomains[0].ID != 201 {
		t.Fatalf("Subdomains = %#v, want one item with ID 201", got.Subdomains)
	}
	if got.Pagination.Page != 2 || got.Pagination.PerPage != 50 || !got.Pagination.HasMore {
		t.Fatalf("Pagination = %#v, want page=2 per_page=50 has_more=true", got.Pagination)
	}
	if got.Pagination.NextPage != 3 || got.Pagination.PrevPage != 1 || got.Pagination.Total != 12500 {
		t.Fatalf("Pagination = %#v, want next=3 prev=1 total=12500", got.Pagination)
	}
}

func TestRegisterSubdomainBuildsRequest(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if got := r.URL.Query().Get("endpoint"); got != "subdomains" {
			t.Fatalf("endpoint = %q, want subdomains", got)
		}
		if got := r.URL.Query().Get("action"); got != "register" {
			t.Fatalf("action = %q, want register", got)
		}

		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body failed: %v", err)
		}
		if payload["subdomain"] != "myapp" {
			t.Fatalf("subdomain = %q, want myapp", payload["subdomain"])
		}
		if payload["rootdomain"] != "example.com" {
			t.Fatalf("rootdomain = %q, want example.com", payload["rootdomain"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"message":"Subdomain registered successfully","subdomain_id":3,"full_domain":"myapp.example.com"}`))
	})

	got, err := client.RegisterSubdomain(context.Background(), RegisterSubdomainRequest{
		Subdomain:  " myapp ",
		Rootdomain: " example.com ",
	})
	if err != nil {
		t.Fatalf("RegisterSubdomain returned error: %v", err)
	}
	if got.SubdomainID != 3 || got.FullDomain != "myapp.example.com" {
		t.Fatalf("result = %#v, want id=3 full_domain=myapp.example.com", got)
	}
}

func TestGetSubdomainBuildsRequest(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if got := r.URL.Query().Get("endpoint"); got != "subdomains" {
			t.Fatalf("endpoint = %q, want subdomains", got)
		}
		if got := r.URL.Query().Get("action"); got != "get" {
			t.Fatalf("action = %q, want get", got)
		}
		if got := r.URL.Query().Get("subdomain_id"); got != "1" {
			t.Fatalf("subdomain_id = %q, want 1", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"subdomain":{"id":1,"subdomain":"test","rootdomain":"example.com","full_domain":"test.example.com","status":"active","created_at":"2025-10-19 10:00:00","updated_at":"2025-10-19 10:00:00"},"dns_records":[{"id":1,"name":"test.example.com","type":"A","content":"192.168.1.1","ttl":600,"priority":null,"status":"active","created_at":"2025-10-19 10:05:00"}],"dns_count":1}`))
	})

	got, err := client.GetSubdomain(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetSubdomain returned error: %v", err)
	}
	if got.Subdomain.ID != 1 || got.Subdomain.FullDomain != "test.example.com" {
		t.Fatalf("Subdomain = %#v, want id=1 full_domain=test.example.com", got.Subdomain)
	}
	if got.DNSCount != 1 || len(got.DNSRecords) != 1 {
		t.Fatalf("DNSCount/Records = %d/%d, want 1/1", got.DNSCount, len(got.DNSRecords))
	}
}

func TestDeleteSubdomainBuildsRequest(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if got := r.URL.Query().Get("endpoint"); got != "subdomains" {
			t.Fatalf("endpoint = %q, want subdomains", got)
		}
		if got := r.URL.Query().Get("action"); got != "delete" {
			t.Fatalf("action = %q, want delete", got)
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body failed: %v", err)
		}
		if payload["subdomain_id"] != float64(1) {
			t.Fatalf("subdomain_id = %v, want 1", payload["subdomain_id"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"message":"Subdomain deleted successfully","subdomain_id":1,"full_domain":"test.example.com","dns_records_deleted":4}`))
	})

	if err := client.DeleteSubdomain(context.Background(), 1); err != nil {
		t.Fatalf("DeleteSubdomain returned error: %v", err)
	}
}

func TestDeleteSubdomainWithResultReturnsDeletedCount(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"message":"Subdomain deleted successfully","subdomain_id":1,"full_domain":"test.example.com","dns_records_deleted":4}`))
	})

	got, err := client.DeleteSubdomainWithResult(context.Background(), 1)
	if err != nil {
		t.Fatalf("DeleteSubdomainWithResult returned error: %v", err)
	}
	if got.SubdomainID != 1 || got.FullDomain != "test.example.com" || got.DNSRecordsDeleted != 4 {
		t.Fatalf("result = %#v, want id=1 full_domain=test.example.com dns_records_deleted=4", got)
	}
}

func TestRenewSubdomainReturnsExtendedFields(t *testing.T) {
	type renewRequest struct {
		SubdomainID int `json:"subdomain_id"`
	}

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if got := r.URL.Query().Get("endpoint"); got != "subdomains" {
			t.Fatalf("endpoint = %q, want subdomains", got)
		}
		if got := r.URL.Query().Get("action"); got != "renew" {
			t.Fatalf("action = %q, want renew", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type = %q, want application/json", got)
		}

		var payload renewRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body failed: %v", err)
		}
		if payload.SubdomainID != 3 {
			t.Fatalf("subdomain_id = %d, want 3", payload.SubdomainID)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success":             true,
			"message":             "Subdomain renewed successfully",
			"subdomain_id":        3,
			"subdomain":           "myapp",
			"previous_expires_at": "2025-05-01 00:00:00",
			"new_expires_at":      "2026-05-01 00:00:00",
			"renewed_at":          "2025-04-10 12:34:56",
			"never_expires":       1,
			"status":              "active",
			"remaining_days":      366,
			"charged_amount":      9.9,
		})
	})

	got, err := client.RenewSubdomain(context.Background(), 3)
	if err != nil {
		t.Fatalf("RenewSubdomain returned error: %v", err)
	}
	if got.Message != "Subdomain renewed successfully" {
		t.Fatalf("Message = %q, want Subdomain renewed successfully", got.Message)
	}
	if got.RenewedAt != "2025-04-10 12:34:56" {
		t.Fatalf("RenewedAt = %q, want 2025-04-10 12:34:56", got.RenewedAt)
	}
	if !got.NeverExpires {
		t.Fatalf("NeverExpires = false, want true")
	}
	if got.Status != "active" {
		t.Fatalf("Status = %q, want active", got.Status)
	}
	if got.ChargedAmount != 9.9 {
		t.Fatalf("ChargedAmount = %v, want 9.9", got.ChargedAmount)
	}
}

func TestListSubdomainsReturnsStructuredHTTPError(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"Rate limit exceeded","limit":60,"remaining":0,"reset_at":"2025-10-19 15:31:00"}`))
	})

	_, err := client.ListSubdomains(context.Background())
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("StatusCode = %d, want %d", apiErr.StatusCode, http.StatusTooManyRequests)
	}
	if apiErr.ErrorText != "Rate limit exceeded" {
		t.Fatalf("ErrorText = %q, want Rate limit exceeded", apiErr.ErrorText)
	}
	if apiErr.Limit == nil || *apiErr.Limit != 60 {
		t.Fatalf("Limit = %v, want 60", apiErr.Limit)
	}
	if apiErr.Remaining == nil || *apiErr.Remaining != 0 {
		t.Fatalf("Remaining = %v, want 0", apiErr.Remaining)
	}
	if apiErr.ResetAt != "2025-10-19 15:31:00" {
		t.Fatalf("ResetAt = %q, want 2025-10-19 15:31:00", apiErr.ResetAt)
	}
}

func TestListSubdomainsReturnsUnifiedErrorDetails(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"success":false,"error_code":"rate_limit_exceeded","message":"Rate limit exceeded","details":{"limit":60,"remaining":0,"reset_at":"2025-10-19 15:31:00","request_id":"req_123"},"error":"Rate limit exceeded"}`))
	})

	_, err := client.ListSubdomains(context.Background())
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.ErrorCode != "rate_limit_exceeded" {
		t.Fatalf("ErrorCode = %q, want rate_limit_exceeded", apiErr.ErrorCode)
	}
	if apiErr.Limit == nil || *apiErr.Limit != 60 {
		t.Fatalf("Limit = %v, want 60", apiErr.Limit)
	}
	if apiErr.Remaining == nil || *apiErr.Remaining != 0 {
		t.Fatalf("Remaining = %v, want 0", apiErr.Remaining)
	}
	if apiErr.ResetAt != "2025-10-19 15:31:00" {
		t.Fatalf("ResetAt = %q, want 2025-10-19 15:31:00", apiErr.ResetAt)
	}
	if apiErr.RequestID != "req_123" {
		t.Fatalf("RequestID = %q, want req_123", apiErr.RequestID)
	}
	if apiErr.Details["request_id"] != "req_123" {
		t.Fatalf("Details[request_id] = %v, want req_123", apiErr.Details["request_id"])
	}
}

func TestListSubdomainsReturnsStructuredBusinessError(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":false,"error":"Invalid API key"}`))
	})

	_, err := client.ListSubdomains(context.Background())
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.Operation != "list subdomains" {
		t.Fatalf("Operation = %q, want list subdomains", apiErr.Operation)
	}
	if apiErr.ErrorText != "Invalid API key" {
		t.Fatalf("ErrorText = %q, want Invalid API key", apiErr.ErrorText)
	}
}

func TestListDNSRecordsDecodesV2Fields(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("endpoint"); got != "dns_records" {
			t.Fatalf("endpoint = %q, want dns_records", got)
		}
		if got := r.URL.Query().Get("action"); got != "list" {
			t.Fatalf("action = %q, want list", got)
		}
		if got := r.URL.Query().Get("subdomain_id"); got != "1" {
			t.Fatalf("subdomain_id = %q, want 1", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"count":1,"records":[{"id":1,"record_id":"5a0ce6c4d1d4c71bc5e60a2a2a0e4997","name":"test.example.com","type":"A","content":"192.168.1.1","ttl":600,"priority":null,"line":"default","proxied":false,"status":"active","created_at":"2025-10-19 10:05:00","updated_at":"2025-10-19 10:05:00"}]}`))
	})

	got, err := client.ListDNSRecords(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListDNSRecords returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0].RecordID != "5a0ce6c4d1d4c71bc5e60a2a2a0e4997" {
		t.Fatalf("RecordID = %q, want provider record id", got[0].RecordID)
	}
	if got[0].Line == nil || *got[0].Line != "default" {
		t.Fatalf("Line = %v, want default", got[0].Line)
	}
}

func TestListDNSRecordsWithResultReturnsCount(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"count":2,"records":[{"id":1,"name":"test.example.com","type":"A","content":"192.168.1.1","ttl":600},{"id":2,"name":"www.test.example.com","type":"CNAME","content":"test.example.com","ttl":600}]}`))
	})

	got, err := client.ListDNSRecordsWithResult(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListDNSRecordsWithResult returned error: %v", err)
	}
	if got.Count != 2 || len(got.Records) != 2 {
		t.Fatalf("result = %#v, want count=2 and two records", got)
	}
}

func TestCreateDNSRecordNormalizesType(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body failed: %v", err)
		}
		if payload["type"] != "TXT" {
			t.Fatalf("type = %v, want TXT", payload["type"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"message":"DNS record created successfully","record_id":5}`))
	})

	result, err := client.CreateDNSRecord(context.Background(), CreateDNSRecordRequest{
		SubdomainID: 1,
		Type:        "txt",
		Content:     "hello",
	})
	if err != nil {
		t.Fatalf("CreateDNSRecord returned error: %v", err)
	}
	if result.RecordID != 5 {
		t.Fatalf("RecordID = %d, want 5", result.RecordID)
	}
}

func TestCreateDNSRecordAllowsMXWithoutPriority(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body failed: %v", err)
		}
		if payload["type"] != "MX" {
			t.Fatalf("type = %v, want MX", payload["type"])
		}
		if _, ok := payload["priority"]; ok {
			t.Fatalf("priority should be omitted when not set, payload=%v", payload)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"message":"DNS record created successfully","id":6,"record_id":"provider-6"}`))
	})

	result, err := client.CreateDNSRecord(context.Background(), CreateDNSRecordRequest{
		SubdomainID: 1,
		Type:        "mx",
		Content:     "mail.example.com",
	})
	if err != nil {
		t.Fatalf("CreateDNSRecord returned error: %v", err)
	}
	if result.ID != 6 || result.RecordID != 6 || result.ProviderRecordID != "provider-6" {
		t.Fatalf("result = %#v, want ID/RecordID 6 and ProviderRecordID provider-6", result)
	}
}

func TestCreateDNSRecordAllowsMXPriority(t *testing.T) {
	priority := 10

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body failed: %v", err)
		}
		if payload["type"] != "MX" {
			t.Fatalf("type = %v, want MX", payload["type"])
		}
		if payload["priority"] != float64(10) {
			t.Fatalf("priority = %v, want 10", payload["priority"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"message":"DNS record created successfully","record_id":6}`))
	})

	result, err := client.CreateDNSRecord(context.Background(), CreateDNSRecordRequest{
		SubdomainID: 1,
		Type:        "mx",
		Content:     "mail.example.com",
		Priority:    &priority,
	})
	if err != nil {
		t.Fatalf("CreateDNSRecord with MX priority returned error: %v", err)
	}
	if result.RecordID != 6 {
		t.Fatalf("RecordID = %d, want 6", result.RecordID)
	}
}

func TestNormalizeRecordTypeSupportsV2Types(t *testing.T) {
	for _, recordType := range []string{"NS", "SRV", "CAA"} {
		got, err := normalizeRecordType(recordType)
		if err != nil {
			t.Fatalf("normalizeRecordType(%q) returned error: %v", recordType, err)
		}
		if got != recordType {
			t.Fatalf("normalizeRecordType(%q) = %q, want %q", recordType, got, recordType)
		}
	}
}

func TestCreateDNSRecordSupportsStructuredSRVFields(t *testing.T) {
	weight := 5
	port := 443
	priority := 10

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body failed: %v", err)
		}
		wantPayload := map[string]any{
			"subdomain_id":  1.0,
			"type":          "SRV",
			"name":          "_https._tcp",
			"priority":      10.0,
			"record_weight": 5.0,
			"record_port":   443.0,
			"record_target": "target.example.com",
			"line":          "default",
		}
		for key, want := range wantPayload {
			if got := payload[key]; got != want {
				t.Fatalf("%s = %v, want %v; payload=%v", key, got, want, payload)
			}
		}
		if _, ok := payload["content"]; ok {
			t.Fatalf("content should be omitted for structured SRV request, payload=%v", payload)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"message":"DNS record created successfully","id":3,"record_id":"5a0ce6c4d1d4c71bc5e60a2a2a0e4997"}`))
	})

	result, err := client.CreateDNSRecord(context.Background(), CreateDNSRecordRequest{
		SubdomainID:  1,
		Type:         "srv",
		Name:         "_https._tcp",
		Priority:     &priority,
		Line:         "default",
		RecordWeight: &weight,
		RecordPort:   &port,
		RecordTarget: "target.example.com",
	})
	if err != nil {
		t.Fatalf("CreateDNSRecord returned error: %v", err)
	}
	if result.ID != 3 || result.RecordID != 3 {
		t.Fatalf("result = %#v, want ID and RecordID 3", result)
	}
	if result.ProviderRecordID != "5a0ce6c4d1d4c71bc5e60a2a2a0e4997" {
		t.Fatalf("ProviderRecordID = %q, want provider record id", result.ProviderRecordID)
	}
}

func TestCreateDNSRecordRejectsConflictingAliases(t *testing.T) {
	weight := 5
	aliasWeight := 6

	client := &Client{}
	_, err := client.CreateDNSRecord(context.Background(), CreateDNSRecordRequest{
		SubdomainID:  1,
		Type:         "SRV",
		Name:         "_https._tcp",
		RecordWeight: &weight,
		Weight:       &aliasWeight,
		RecordPort:   intPtr(443),
		RecordTarget: "target.example.com",
	})
	if err == nil {
		t.Fatalf("expected conflicting alias error, got nil")
	}
}

func TestCreateDNSRecordAllowsMatchingAliases(t *testing.T) {
	weight := 5
	aliasWeight := 5

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body failed: %v", err)
		}
		if payload["record_weight"] != float64(5) {
			t.Fatalf("record_weight = %v, want 5", payload["record_weight"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"message":"DNS record created successfully","id":3,"record_id":"provider-3"}`))
	})

	_, err := client.CreateDNSRecord(context.Background(), CreateDNSRecordRequest{
		SubdomainID:  1,
		Type:         "SRV",
		Name:         "_https._tcp",
		RecordWeight: &weight,
		Weight:       &aliasWeight,
		RecordPort:   intPtr(443),
		RecordTarget: "target.example.com",
	})
	if err != nil {
		t.Fatalf("CreateDNSRecord returned error: %v", err)
	}
}

func TestUpdateDNSRecordWithResultSupportsV2Fields(t *testing.T) {
	recordType := "CAA"
	name := "@"
	caaFlag := 0
	caaTag := "issue"
	caaValue := "letsencrypt.org"

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("endpoint"); got != "dns_records" {
			t.Fatalf("endpoint = %q, want dns_records", got)
		}
		if got := r.URL.Query().Get("action"); got != "update" {
			t.Fatalf("action = %q, want update", got)
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body failed: %v", err)
		}
		wantPayload := map[string]any{
			"id":        1.0,
			"type":      "CAA",
			"name":      "@",
			"caa_flag":  0.0,
			"caa_tag":   "issue",
			"caa_value": "letsencrypt.org",
		}
		for key, want := range wantPayload {
			if got := payload[key]; got != want {
				t.Fatalf("%s = %v, want %v; payload=%v", key, got, want, payload)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"message":"DNS record updated successfully","id":1,"record_id":"provider-1"}`))
	})

	got, err := client.UpdateDNSRecordWithResult(context.Background(), UpdateDNSRecordRequest{
		ID:       1,
		Type:     &recordType,
		Name:     &name,
		CAAFlag:  &caaFlag,
		CAATag:   &caaTag,
		CAAValue: &caaValue,
	})
	if err != nil {
		t.Fatalf("UpdateDNSRecordWithResult returned error: %v", err)
	}
	if got.ID != 1 || got.RecordID != 1 || got.ProviderRecordID != "provider-1" {
		t.Fatalf("result = %#v, want ID/RecordID 1 and ProviderRecordID provider-1", got)
	}
}

func TestUpdateDNSRecordRejectsConflictingAliases(t *testing.T) {
	recordTarget := "target.example.com"
	aliasTarget := "other.example.com"

	client := &Client{}
	_, err := client.UpdateDNSRecordWithResult(context.Background(), UpdateDNSRecordRequest{
		ID:           1,
		RecordTarget: &recordTarget,
		Target:       &aliasTarget,
	})
	if err == nil {
		t.Fatalf("expected conflicting alias error, got nil")
	}
}

func TestUpdateDNSRecordFallsBackFromBlankProviderRecordID(t *testing.T) {
	content := "192.168.1.2"

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body failed: %v", err)
		}
		if payload["record_id"] != float64(9) {
			t.Fatalf("record_id = %v, want 9; payload=%v", payload["record_id"], payload)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"message":"DNS record updated successfully","id":9,"record_id":"provider-9"}`))
	})

	_, err := client.UpdateDNSRecordWithResult(context.Background(), UpdateDNSRecordRequest{
		RecordID:         9,
		ProviderRecordID: " ",
		Content:          &content,
	})
	if err != nil {
		t.Fatalf("UpdateDNSRecordWithResult returned error: %v", err)
	}
}

func TestDeleteDNSRecordUsesInternalID(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body failed: %v", err)
		}
		if payload["id"] != float64(5) {
			t.Fatalf("id = %v, want 5; payload=%v", payload["id"], payload)
		}
		if _, ok := payload["record_id"]; ok {
			t.Fatalf("record_id should be omitted for internal id delete, payload=%v", payload)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"message":"DNS record deleted successfully"}`))
	})

	if err := client.DeleteDNSRecord(context.Background(), 5); err != nil {
		t.Fatalf("DeleteDNSRecord returned error: %v", err)
	}
}

func intPtr(value int) *int {
	return &value
}

func TestDeleteDNSRecordByIDReturnsMessage(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"message":"DNS record deleted successfully"}`))
	})

	got, err := client.DeleteDNSRecordByID(context.Background(), 5)
	if err != nil {
		t.Fatalf("DeleteDNSRecordByID returned error: %v", err)
	}
	if got.Message != "DNS record deleted successfully" {
		t.Fatalf("Message = %q, want DNS record deleted successfully", got.Message)
	}
}

func TestDeleteDNSRecordByProviderRecordID(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body failed: %v", err)
		}
		if payload["record_id"] != "5a0ce6c4d1d4c71bc5e60a2a2a0e4997" {
			t.Fatalf("record_id = %q, want provider record id", payload["record_id"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"message":"DNS record deleted successfully"}`))
	})

	if err := client.DeleteDNSRecordByProviderRecordID(context.Background(), "5a0ce6c4d1d4c71bc5e60a2a2a0e4997"); err != nil {
		t.Fatalf("DeleteDNSRecordByProviderRecordID returned error: %v", err)
	}
}

func TestAPIKeyManagementMethodsBuildRequests(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("endpoint"); got != "keys" {
			t.Fatalf("endpoint = %q, want keys", got)
		}

		w.Header().Set("Content-Type", "application/json")
		switch action := r.URL.Query().Get("action"); action {
		case "list":
			if r.Method != http.MethodGet {
				t.Fatalf("method = %s, want GET", r.Method)
			}
			_, _ = w.Write([]byte(`{"success":true,"count":2,"keys":[{"id":1,"key_name":"Production Environment Key","api_key":"cfsd_xxxxxxxxxx","status":"active","request_count":1523,"last_used_at":"2025-10-19 15:30:00","created_at":"2025-10-19 10:00:00"},{"id":2,"key_name":"test-key","api_key":"cfsd_yyyyyyyyyy","status":"active","request_count":45,"last_used_at":"2025-10-19 14:00:00","created_at":"2025-10-19 11:00:00"}]}`))
		case "create":
			var payload map[string]string
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode create request body failed: %v", err)
			}
			if payload["key_name"] != "New API Key" {
				t.Fatalf("key_name = %q, want New API Key", payload["key_name"])
			}
			if payload["ip_whitelist"] != "192.168.1.1,192.168.1.2" {
				t.Fatalf("ip_whitelist = %q, want whitelist", payload["ip_whitelist"])
			}
			_, _ = w.Write([]byte(`{"success":true,"message":"API key created successfully","api_key":"cfsd_zzzzzzzzzz","api_secret":"aaaaaaaaaaaaaaaa","warning":"Please save the api_secret, it will not be shown again"}`))
		case "delete":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode delete request body failed: %v", err)
			}
			if payload["key_id"] != float64(2) {
				t.Fatalf("key_id = %v, want 2", payload["key_id"])
			}
			_, _ = w.Write([]byte(`{"success":true,"message":"API key deleted successfully"}`))
		case "regenerate":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode regenerate request body failed: %v", err)
			}
			if payload["key_id"] != float64(1) {
				t.Fatalf("key_id = %v, want 1", payload["key_id"])
			}
			_, _ = w.Write([]byte(`{"success":true,"message":"API secret regenerated successfully","api_key":"cfsd_xxxxxxxxxx","api_secret":"new_secret_here","warning":"Please save the new api_secret, it will not be shown again"}`))
		default:
			t.Fatalf("unexpected action %q", action)
		}
	})

	keys, err := client.ListAPIKeys(context.Background())
	if err != nil {
		t.Fatalf("ListAPIKeys returned error: %v", err)
	}
	if len(keys) != 2 || keys[0].RequestCount != 1523 {
		t.Fatalf("keys = %#v, want two keys with request count 1523", keys)
	}

	created, err := client.CreateAPIKey(context.Background(), CreateAPIKeyRequest{
		KeyName:     " New API Key ",
		IPWhitelist: " 192.168.1.1,192.168.1.2 ",
	})
	if err != nil {
		t.Fatalf("CreateAPIKey returned error: %v", err)
	}
	if created.APIKey != "cfsd_zzzzzzzzzz" || created.APISecret != "aaaaaaaaaaaaaaaa" {
		t.Fatalf("created = %#v, want api key and secret", created)
	}

	if err := client.DeleteAPIKey(context.Background(), 2); err != nil {
		t.Fatalf("DeleteAPIKey returned error: %v", err)
	}

	regenerated, err := client.RegenerateAPIKey(context.Background(), 1)
	if err != nil {
		t.Fatalf("RegenerateAPIKey returned error: %v", err)
	}
	if regenerated.APISecret != "new_secret_here" {
		t.Fatalf("APISecret = %q, want new_secret_here", regenerated.APISecret)
	}
}

func TestListAPIKeysWithResultReturnsCount(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"count":1,"keys":[{"id":1,"key_name":"Production Environment Key","api_key":"cfsd_xxxxxxxxxx","status":"active","request_count":1523,"last_used_at":"2025-10-19 15:30:00","created_at":"2025-10-19 10:00:00"}]}`))
	})

	got, err := client.ListAPIKeysWithResult(context.Background())
	if err != nil {
		t.Fatalf("ListAPIKeysWithResult returned error: %v", err)
	}
	if got.Count != 1 || len(got.Keys) != 1 {
		t.Fatalf("result = %#v, want count=1 and one key", got)
	}
}

func TestGetQuotaBuildsRequest(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if got := r.URL.Query().Get("endpoint"); got != "quota" {
			t.Fatalf("endpoint = %q, want quota", got)
		}
		if got := r.URL.Query().Get("action"); got != "" {
			t.Fatalf("action = %q, want empty", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"quota":{"used":3,"base":5,"invite_bonus":2,"total":7,"available":4}}`))
	})

	got, err := client.GetQuota(context.Background())
	if err != nil {
		t.Fatalf("GetQuota returned error: %v", err)
	}
	if got.Used != 3 || got.Base != 5 || got.InviteBonus != 2 || got.Total != 7 || got.Available != 4 {
		t.Fatalf("Quota = %#v, want documented quota values", got)
	}
}

func TestWhoisPublicClientBuildsUnauthenticatedRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("endpoint"); got != "whois" {
			t.Fatalf("endpoint = %q, want whois", got)
		}
		if got := r.URL.Query().Get("action"); got != "" {
			t.Fatalf("action = %q, want empty", got)
		}
		if got := r.URL.Query().Get("domain"); got != "foo.example.com" {
			t.Fatalf("domain = %q, want foo.example.com", got)
		}
		if got := r.Header.Get("X-API-Key"); got != "" {
			t.Fatalf("X-API-Key = %q, want empty", got)
		}
		if got := r.Header.Get("X-API-Secret"); got != "" {
			t.Fatalf("X-API-Secret = %q, want empty", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"domain":"foo.example.com","status":"active","registered_at":"2025-01-10 08:30:00","expires_at":"2026-01-10 08:30:00","registrant_email":"whois@example.com","nameservers":["ns1.example.net","ns2.example.net"],"name_servers":["ns1.example.net","ns2.example.net"],"rate_limit":{"limit":2,"remaining":1,"reset_at":"2025-01-10 08:31:00"}}`))
	}))
	defer server.Close()

	client := NewPublicClient(PublicConfig{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})

	got, err := client.Whois(context.Background(), "foo.example.com")
	if err != nil {
		t.Fatalf("Whois returned error: %v", err)
	}
	if !got.Registered {
		t.Fatalf("Registered = false, want true")
	}
	if got.Domain != "foo.example.com" || got.Status != "active" {
		t.Fatalf("Whois result = %#v, want foo.example.com active", got)
	}
	if len(got.Nameservers) != 2 || got.Nameservers[0] != "ns1.example.net" {
		t.Fatalf("Nameservers = %#v, want two nameservers", got.Nameservers)
	}
	if got.RateLimit == nil || got.RateLimit.Limit != 2 || got.RateLimit.Remaining != 1 {
		t.Fatalf("RateLimit = %#v, want limit=2 remaining=1", got.RateLimit)
	}
}

func TestWhoisDecodesUnregisteredDomain(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-API-Key"); got != "" {
			t.Fatalf("X-API-Key = %q, want empty", got)
		}
		if got := r.URL.Query().Get("domain"); got != "missing.example.com" {
			t.Fatalf("domain = %q, want missing.example.com", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"domain":"missing.example.com","registered":false,"status":"unregistered","message":"domain not registered"}`))
	}))
	defer server.Close()

	client := NewPublicClient(PublicConfig{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})

	got, err := client.Whois(context.Background(), "missing.example.com")
	if err != nil {
		t.Fatalf("Whois returned error: %v", err)
	}
	if got.Registered {
		t.Fatalf("Registered = true, want false")
	}
	if got.Status != "unregistered" || got.Message != "domain not registered" {
		t.Fatalf("result = %#v, want unregistered message", got)
	}
}

func TestWhoisFallsBackNameserversFromNameServers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"domain":"foo.example.com","status":"active","name_servers":["ns1.example.net","ns2.example.net"]}`))
	}))
	defer server.Close()

	client := NewPublicClient(PublicConfig{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})

	got, err := client.Whois(context.Background(), "foo.example.com")
	if err != nil {
		t.Fatalf("Whois returned error: %v", err)
	}
	if len(got.Nameservers) != 2 || got.Nameservers[0] != "ns1.example.net" {
		t.Fatalf("Nameservers = %#v, want fallback from name_servers", got.Nameservers)
	}
}

func TestPermanentUpgradeMethodsBuildRequests(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		action := r.URL.Query().Get("action")
		if got := r.URL.Query().Get("endpoint"); got != "permanent_upgrade" {
			t.Fatalf("endpoint = %q, want permanent_upgrade", got)
		}

		w.Header().Set("Content-Type", "application/json")
		switch action {
		case "list":
			if r.Method != http.MethodGet {
				t.Fatalf("method = %s, want GET", r.Method)
			}
			if got := r.URL.Query().Get("page"); got != "1" {
				t.Fatalf("page = %q, want 1", got)
			}
			if got := r.URL.Query().Get("per_page"); got != "10" {
				t.Fatalf("per_page = %q, want 10", got)
			}
			_, _ = w.Write([]byte(`{"success":true,"state":{"requests":[{"request_id":456,"subdomain_id":123,"status":"pending","assist_code":"ABCD1234"}],"assist_logs":[{"assist_code":"ABCD1234"}],"eligible_subdomains":[{"id":123,"full_domain":"foo.example.com"}]}}`))
		case "create":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode create request body failed: %v", err)
			}
			if payload["subdomain_id"] != float64(123) {
				t.Fatalf("subdomain_id = %v, want 123", payload["subdomain_id"])
			}
			_, _ = w.Write([]byte(`{"success":true,"message":"created","request_id":456,"assist_code":"ABCD1234"}`))
		case "assist":
			var payload map[string]string
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode assist request body failed: %v", err)
			}
			if payload["assist_code"] != "ABCD1234" {
				t.Fatalf("assist_code = %q, want ABCD1234", payload["assist_code"])
			}
			_, _ = w.Write([]byte(`{"success":true,"message":"assisted"}`))
		case "cancel":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode cancel request body failed: %v", err)
			}
			if payload["request_id"] != float64(456) {
				t.Fatalf("request_id = %v, want 456", payload["request_id"])
			}
			_, _ = w.Write([]byte(`{"success":true,"message":"cancelled","request_id":456}`))
		default:
			t.Fatalf("unexpected action %q", action)
		}
	})

	state, err := client.ListPermanentUpgrades(context.Background(), PermanentUpgradeListOptions{
		Page:    1,
		PerPage: 10,
	})
	if err != nil {
		t.Fatalf("ListPermanentUpgrades returned error: %v", err)
	}
	if len(state.Requests) != 1 || state.Requests[0].RequestID != 456 {
		t.Fatalf("Requests = %#v, want request_id 456", state.Requests)
	}
	if len(state.EligibleSubdomains) != 1 || state.EligibleSubdomains[0].ID != 123 {
		t.Fatalf("EligibleSubdomains = %#v, want subdomain id 123", state.EligibleSubdomains)
	}

	created, err := client.CreatePermanentUpgrade(context.Background(), 123)
	if err != nil {
		t.Fatalf("CreatePermanentUpgrade returned error: %v", err)
	}
	if created.RequestID != 456 || created.AssistCode != "ABCD1234" {
		t.Fatalf("created = %#v, want request id 456 and assist code ABCD1234", created)
	}

	if _, err := client.AssistPermanentUpgrade(context.Background(), "ABCD1234"); err != nil {
		t.Fatalf("AssistPermanentUpgrade returned error: %v", err)
	}
	if _, err := client.CancelPermanentUpgrade(context.Background(), 456); err != nil {
		t.Fatalf("CancelPermanentUpgrade returned error: %v", err)
	}
}

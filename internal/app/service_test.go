package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"dnsherene/pkg/dnshe"
)

func TestRunNoSubdomainsIsNotAnError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("action"); got == "renew" {
			t.Fatalf("unexpected renew request when subdomain list is empty")
		}

		if got := r.URL.Query().Get("endpoint"); got != "subdomains" {
			t.Fatalf("endpoint = %q, want subdomains", got)
		}
		if got := r.URL.Query().Get("action"); got != "list" {
			t.Fatalf("action = %q, want list", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success":    true,
			"count":      0,
			"subdomains": []map[string]any{},
		})
	}))
	defer server.Close()

	client, err := dnshe.NewClient(dnshe.Config{
		BaseURL:    server.URL,
		APIKey:     "test-key",
		APISecret:  "test-secret",
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	service, err := NewService(client)
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}

	result, err := service.Run(context.Background(), false)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Matched != 0 || result.Renewed != 0 || result.Failed != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(result.Domains) != 0 {
		t.Fatalf("unexpected domains: %+v", result.Domains)
	}
}

func TestRunSkipsSubdomainsOutsideRenewWindow(t *testing.T) {
	now := time.Now().UTC()
	renewCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("action"); got == "renew" {
			renewCalled = true
			t.Fatalf("unexpected renew request for subdomain outside renew window")
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
					"created_at":  now.Add(-10 * 24 * time.Hour).Format("2006-01-02 15:04:05"),
				},
			},
		})
	}))
	defer server.Close()

	client, err := dnshe.NewClient(dnshe.Config{
		BaseURL:    server.URL,
		APIKey:     "test-key",
		APISecret:  "test-secret",
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	service, err := NewService(client)
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}

	result, err := service.Run(context.Background(), false)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if renewCalled {
		t.Fatalf("renew request should not be called")
	}
	if result.Matched != 0 || result.Renewed != 0 || result.Failed != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(result.Domains) != 1 {
		t.Fatalf("domains len = %d, want 1", len(result.Domains))
	}
	if result.Domains[0].Domain != "api.example.com" {
		t.Fatalf("domain = %q, want api.example.com", result.Domains[0].Domain)
	}
	if result.Domains[0].ExpiresAt == "" {
		t.Fatalf("expires_at should not be empty")
	}
}

func TestRunRenewsSubdomainsWithin180Days(t *testing.T) {
	now := time.Now().UTC()
	renewCalls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		action := r.URL.Query().Get("action")
		w.Header().Set("Content-Type", "application/json")

		switch action {
		case "list":
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
						"created_at":  now.Add(-220 * 24 * time.Hour).Format("2006-01-02 15:04:05"),
					},
				},
			})
		case "renew":
			renewCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success":             true,
				"message":             "Subdomain renewed successfully",
				"subdomain_id":        1,
				"subdomain":           "api.example.com",
				"previous_expires_at": now.Add(145 * 24 * time.Hour).Format("2006-01-02 15:04:05"),
				"new_expires_at":      now.Add(510 * 24 * time.Hour).Format("2006-01-02 15:04:05"),
				"renewed_at":          now.Format("2006-01-02 15:04:05"),
				"never_expires":       0,
				"status":              "active",
				"remaining_days":      365,
			})
		default:
			t.Fatalf("unexpected action: %q", action)
		}
	}))
	defer server.Close()

	client, err := dnshe.NewClient(dnshe.Config{
		BaseURL:    server.URL,
		APIKey:     "test-key",
		APISecret:  "test-secret",
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	service, err := NewService(client)
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}

	result, err := service.Run(context.Background(), false)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if renewCalls != 1 {
		t.Fatalf("renew calls = %d, want 1", renewCalls)
	}
	if result.Matched != 1 || result.Renewed != 1 || result.Failed != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(result.Domains) != 1 || result.Domains[0].ExpiresAt == "" {
		t.Fatalf("unexpected domains: %+v", result.Domains)
	}
}

func TestRunUsesRemainingDaysWhenPresent(t *testing.T) {
	renewCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("action"); got == "renew" {
			renewCalled = true
			t.Fatalf("unexpected renew request when remaining_days >= 180")
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"count":   1,
			"subdomains": []map[string]any{
				{
					"id":             1,
					"subdomain":      "api",
					"rootdomain":     "example.com",
					"full_domain":    "api.example.com",
					"status":         "active",
					"created_at":     time.Now().UTC().Add(-300 * 24 * time.Hour).Format("2006-01-02 15:04:05"),
					"remaining_days": 200,
				},
			},
		})
	}))
	defer server.Close()

	client, err := dnshe.NewClient(dnshe.Config{
		BaseURL:    server.URL,
		APIKey:     "test-key",
		APISecret:  "test-secret",
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	service, err := NewService(client)
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}

	result, err := service.Run(context.Background(), false)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if renewCalled {
		t.Fatalf("renew request should not be called")
	}
	if result.Matched != 0 || result.Renewed != 0 || result.Failed != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(result.Domains) != 1 {
		t.Fatalf("domains len = %d, want 1", len(result.Domains))
	}
	if result.Domains[0].RemainingDays == nil || *result.Domains[0].RemainingDays != 200 {
		t.Fatalf("remaining_days = %v, want 200", result.Domains[0].RemainingDays)
	}
}

func TestFarFutureExpiryReturnsRealRemainingDays(t *testing.T) {
	now := time.Date(2026, time.May, 30, 12, 0, 0, 0, time.UTC)
	target := dnshe.Subdomain{
		ID:        1,
		ExpiresAt: "2999-12-31 23:59:59",
	}

	if shouldRenewSubdomain(now, target) {
		t.Fatalf("%s should not enter renew window", target.ExpiresAt)
	}
	if got := resolveSubdomainExpiry(target); got != target.ExpiresAt {
		t.Fatalf("expires_at = %q, want %q", got, target.ExpiresAt)
	}
	got := resolveRemainingDays(now, target)
	if got == nil {
		t.Fatalf("remaining_days = nil, want real days")
	}
	if *got != 355596 {
		t.Fatalf("remaining_days = %d, want 355596", *got)
	}
}

func TestDaysUntilDoesNotSaturateForFarFutureDate(t *testing.T) {
	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	target := time.Date(2998, time.January, 1, 0, 0, 0, 0, time.UTC)

	got := daysUntil(now, target)
	if got == nil {
		t.Fatalf("daysUntil returned nil")
	}
	if *got <= 106751 {
		t.Fatalf("daysUntil = %d, want greater than saturated duration days", *got)
	}
}

func TestRunTreatsRenewNotYetAvailableAsSkip(t *testing.T) {
	now := time.Now().UTC()
	renewCalls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		action := r.URL.Query().Get("action")
		w.Header().Set("Content-Type", "application/json")

		switch action {
		case "list":
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
						"created_at":  now.Add(-220 * 24 * time.Hour).Format("2006-01-02 15:04:05"),
					},
				},
			})
		case "renew":
			renewCalls++
			w.WriteHeader(http.StatusUnprocessableEntity)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": "renewal not yet available",
			})
		default:
			t.Fatalf("unexpected action: %q", action)
		}
	}))
	defer server.Close()

	client, err := dnshe.NewClient(dnshe.Config{
		BaseURL:    server.URL,
		APIKey:     "test-key",
		APISecret:  "test-secret",
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	service, err := NewService(client)
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}

	result, err := service.Run(context.Background(), false)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if renewCalls != 1 {
		t.Fatalf("renew calls = %d, want 1", renewCalls)
	}
	if result.Matched != 1 || result.Renewed != 0 || result.Failed != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(result.Domains) != 1 {
		t.Fatalf("domains len = %d, want 1", len(result.Domains))
	}
}

func TestRunRetriesListWhenRateLimited(t *testing.T) {
	listCalls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		action := r.URL.Query().Get("action")
		w.Header().Set("Content-Type", "application/json")

		if action != "list" {
			t.Fatalf("unexpected action: %q", action)
		}

		listCalls++
		if listCalls == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error_code": "rate_limit_exceeded",
				"message":    "Rate limit exceeded",
				"details": map[string]any{
					"reset_at": time.Now().UTC().Add(10 * time.Millisecond).Format(time.RFC3339Nano),
				},
			})
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"success":    true,
			"count":      0,
			"subdomains": []map[string]any{},
		})
	}))
	defer server.Close()

	client, err := dnshe.NewClient(dnshe.Config{
		BaseURL:    server.URL,
		APIKey:     "test-key",
		APISecret:  "test-secret",
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	service, err := NewService(client)
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}

	result, err := service.Run(context.Background(), false)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if listCalls != 2 {
		t.Fatalf("list calls = %d, want 2", listCalls)
	}
	if result.Matched != 0 || result.Renewed != 0 || result.Failed != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestRunRetriesRenewWhenRateLimited(t *testing.T) {
	now := time.Now().UTC()
	renewCalls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		action := r.URL.Query().Get("action")
		w.Header().Set("Content-Type", "application/json")

		switch action {
		case "list":
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
						"created_at":  now.Add(-220 * 24 * time.Hour).Format("2006-01-02 15:04:05"),
					},
				},
			})
		case "renew":
			renewCalls++
			if renewCalls == 1 {
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"error_code": "rate_limit_exceeded",
					"message":    "Rate limit exceeded",
					"details": map[string]any{
						"reset_at": time.Now().UTC().Add(10 * time.Millisecond).Format(time.RFC3339Nano),
					},
				})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success":             true,
				"message":             "Subdomain renewed successfully",
				"subdomain_id":        1,
				"subdomain":           "api.example.com",
				"previous_expires_at": now.Add(145 * 24 * time.Hour).Format("2006-01-02 15:04:05"),
				"new_expires_at":      now.Add(510 * 24 * time.Hour).Format("2006-01-02 15:04:05"),
				"renewed_at":          now.Format("2006-01-02 15:04:05"),
				"remaining_days":      365,
			})
		default:
			t.Fatalf("unexpected action: %q", action)
		}
	}))
	defer server.Close()

	client, err := dnshe.NewClient(dnshe.Config{
		BaseURL:    server.URL,
		APIKey:     "test-key",
		APISecret:  "test-secret",
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	service, err := NewService(client)
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}

	result, err := service.Run(context.Background(), false)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if renewCalls != 2 {
		t.Fatalf("renew calls = %d, want 2", renewCalls)
	}
	if result.Renewed != 1 || result.Failed != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestRunStopsRetryingAfterRateLimitRetriesAreExhausted(t *testing.T) {
	now := time.Now().UTC()
	renewCalls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		action := r.URL.Query().Get("action")
		w.Header().Set("Content-Type", "application/json")

		switch action {
		case "list":
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
						"created_at":  now.Add(-220 * 24 * time.Hour).Format("2006-01-02 15:04:05"),
					},
				},
			})
		case "renew":
			renewCalls++
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error_code": "rate_limit_exceeded",
				"message":    "Rate limit exceeded",
				"details": map[string]any{
					"reset_at": time.Now().UTC().Add(10 * time.Millisecond).Format(time.RFC3339Nano),
				},
			})
		default:
			t.Fatalf("unexpected action: %q", action)
		}
	}))
	defer server.Close()

	client, err := dnshe.NewClient(dnshe.Config{
		BaseURL:    server.URL,
		APIKey:     "test-key",
		APISecret:  "test-secret",
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	service, err := NewService(client)
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}

	result, err := service.Run(context.Background(), false)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if renewCalls != maxRateLimitRetries+1 {
		t.Fatalf("renew calls = %d, want %d", renewCalls, maxRateLimitRetries+1)
	}
	if result.Renewed != 0 || result.Failed != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

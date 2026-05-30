package config

import (
	"testing"
	"time"
)

func TestResolveCredentialsSingleItem(t *testing.T) {
	creds, err := resolveCredentials("key-1", "secret-1")
	if err != nil {
		t.Fatalf("resolveCredentials returned error: %v", err)
	}
	if len(creds) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(creds))
	}
	if creds[0].APIKey != "key-1" || creds[0].APISecret != "secret-1" {
		t.Fatalf("unexpected credential: %+v", creds[0])
	}
}

func TestResolveCredentialsMulti(t *testing.T) {
	creds, err := resolveCredentials("k1,k2", "s1,s2")
	if err != nil {
		t.Fatalf("resolveCredentials returned error: %v", err)
	}
	if len(creds) != 2 {
		t.Fatalf("expected 2 credentials, got %d", len(creds))
	}
	if creds[0].APIKey != "k1" || creds[0].APISecret != "s1" {
		t.Fatalf("unexpected credential[0]: %+v", creds[0])
	}
	if creds[1].APIKey != "k2" || creds[1].APISecret != "s2" {
		t.Fatalf("unexpected credential[1]: %+v", creds[1])
	}
}

func TestResolveCredentialsMismatch(t *testing.T) {
	_, err := resolveCredentials("k1,k2", "s1")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestResolveCredentialsMissingSecrets(t *testing.T) {
	_, err := resolveCredentials("k1,k2", "")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestResolveCredentialsMissingKeys(t *testing.T) {
	_, err := resolveCredentials("", "s1,s2")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestSplitList(t *testing.T) {
	got := splitList("k1,\nk2; k3\tk4")
	if len(got) != 4 {
		t.Fatalf("expected 4 items, got %d (%v)", len(got), got)
	}
}

func TestLoadWithLookup(t *testing.T) {
	values := map[string]string{
		"DNSHE_API_KEYS":    "k1,k2",
		"DNSHE_API_SECRETS": "s1,s2",
		"DNSHE_DRY_RUN":     "true",
	}

	cfg, err := loadWithLookup(func(key string) string {
		return values[key]
	})
	if err != nil {
		t.Fatalf("loadWithLookup returned error: %v", err)
	}
	if len(cfg.Credentials) != 2 {
		t.Fatalf("expected 2 credentials, got %d", len(cfg.Credentials))
	}
	if !cfg.DryRun {
		t.Fatalf("expected DryRun=true")
	}
	if cfg.RunTimeout != DefaultRunTimeout {
		t.Fatalf("RunTimeout = %v, want %v", cfg.RunTimeout, DefaultRunTimeout)
	}
}

func TestLoadWithLookupRunTimeout(t *testing.T) {
	values := map[string]string{
		"DNSHE_API_KEYS":    "k1",
		"DNSHE_API_SECRETS": "s1",
		"DNSHE_RUN_TIMEOUT": "2m30s",
	}

	cfg, err := loadWithLookup(func(key string) string {
		return values[key]
	})
	if err != nil {
		t.Fatalf("loadWithLookup returned error: %v", err)
	}
	if cfg.RunTimeout != 150*time.Second {
		t.Fatalf("RunTimeout = %v, want 2m30s", cfg.RunTimeout)
	}
}

func TestLoadWithLookupInvalidRunTimeout(t *testing.T) {
	values := map[string]string{
		"DNSHE_API_KEYS":    "k1",
		"DNSHE_API_SECRETS": "s1",
		"DNSHE_RUN_TIMEOUT": "soon",
	}

	_, err := loadWithLookup(func(key string) string {
		return values[key]
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

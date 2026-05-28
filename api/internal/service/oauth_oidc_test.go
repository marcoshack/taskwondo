package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOIDCDiscovery(t *testing.T) {
	discovery := map[string]string{
		"authorization_endpoint": "https://auth.example.com/auth",
		"token_endpoint":         "https://auth.example.com/token",
		"userinfo_endpoint":      "https://auth.example.com/userinfo",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/openid-configuration" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(discovery)
	}))
	defer server.Close()

	provider := NewOIDCProvider("client-id", "client-secret", "http://localhost/callback", server.URL, server.Client())

	disc, err := provider.discover()
	if err != nil {
		t.Fatalf("discover() error: %v", err)
	}

	if disc.AuthorizationEndpoint != discovery["authorization_endpoint"] {
		t.Errorf("authorization_endpoint = %q, want %q", disc.AuthorizationEndpoint, discovery["authorization_endpoint"])
	}
	if disc.TokenEndpoint != discovery["token_endpoint"] {
		t.Errorf("token_endpoint = %q, want %q", disc.TokenEndpoint, discovery["token_endpoint"])
	}
	if disc.UserinfoEndpoint != discovery["userinfo_endpoint"] {
		t.Errorf("userinfo_endpoint = %q, want %q", disc.UserinfoEndpoint, discovery["userinfo_endpoint"])
	}
}

func TestOIDCDiscoveryMissingEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"authorization_endpoint": "https://auth.example.com/auth",
			// missing token_endpoint and userinfo_endpoint
		})
	}))
	defer server.Close()

	provider := NewOIDCProvider("client-id", "client-secret", "http://localhost/callback", server.URL, server.Client())

	_, err := provider.discover()
	if err == nil {
		t.Fatal("expected error for missing endpoints")
	}
}

func TestOIDCAuthURL(t *testing.T) {
	discovery := map[string]string{
		"authorization_endpoint": "https://auth.example.com/auth",
		"token_endpoint":         "https://auth.example.com/token",
		"userinfo_endpoint":      "https://auth.example.com/userinfo",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(discovery)
	}))
	defer server.Close()

	provider := NewOIDCProvider("my-client", "my-secret", "http://localhost/callback", server.URL, server.Client())

	authURL := provider.AuthURL("test-state")

	if authURL == "" {
		t.Fatal("AuthURL returned empty string")
	}

	// Verify it starts with the discovered authorization endpoint
	if len(authURL) < len(discovery["authorization_endpoint"]) ||
		authURL[:len(discovery["authorization_endpoint"])] != discovery["authorization_endpoint"] {
		t.Errorf("AuthURL should start with %q, got %q", discovery["authorization_endpoint"], authURL)
	}

	// Check required params are present
	for _, param := range []string{"client_id=my-client", "response_type=code", "scope=openid", "state=test-state"} {
		if !containsSubstr(authURL, param) {
			t.Errorf("AuthURL missing param %q in %q", param, authURL)
		}
	}
}

func TestOIDCName(t *testing.T) {
	provider := NewOIDCProvider("id", "secret", "uri", "https://example.com", nil)
	if provider.Name() != "oidc" {
		t.Errorf("Name() = %q, want %q", provider.Name(), "oidc")
	}
}

func TestOIDCExchangeCode(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"authorization_endpoint": "http://not-used/auth",
			"token_endpoint":         "", // will be replaced
			"userinfo_endpoint":      "", // will be replaced
		})
	})

	// We need the server URL for the endpoints, so create a server first then update
	server := httptest.NewServer(mux)
	defer server.Close()

	// Re-register with proper endpoints using server URL
	mux2 := http.NewServeMux()
	mux2.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"authorization_endpoint": server.URL + "/auth",
			"token_endpoint":         server.URL + "/token",
			"userinfo_endpoint":      server.URL + "/userinfo",
		})
	})

	mux2.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("token: expected POST, got %s", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.FormValue("code") != "test-code" {
			t.Errorf("token: code = %q, want %q", r.FormValue("code"), "test-code")
		}
		if r.FormValue("grant_type") != "authorization_code" {
			t.Errorf("token: grant_type = %q", r.FormValue("grant_type"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"access_token": "test-access-token",
			"token_type":   "Bearer",
		})
	})

	mux2.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-access-token" {
			t.Errorf("userinfo: bad authorization header: %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"sub":                "user-123",
			"email":              "user@example.com",
			"email_verified":     true,
			"name":               "Test User",
			"preferred_username": "testuser",
			"picture":            "https://example.com/avatar.jpg",
		})
	})

	// Replace the server handler
	server.Config.Handler = mux2

	provider := NewOIDCProvider("client-id", "client-secret", "http://localhost/callback", server.URL, server.Client())

	info, err := provider.ExchangeCode(context.Background(), "test-code")
	if err != nil {
		t.Fatalf("ExchangeCode() error: %v", err)
	}

	if info.ProviderUserID != "user-123" {
		t.Errorf("ProviderUserID = %q, want %q", info.ProviderUserID, "user-123")
	}
	if info.Email != "user@example.com" {
		t.Errorf("Email = %q, want %q", info.Email, "user@example.com")
	}
	if !info.EmailVerified {
		t.Error("EmailVerified = false, want true")
	}
	if info.DisplayName != "Test User" {
		t.Errorf("DisplayName = %q, want %q", info.DisplayName, "Test User")
	}
	if info.Username != "testuser" {
		t.Errorf("Username = %q, want %q", info.Username, "testuser")
	}
	if info.AvatarURL != "https://example.com/avatar.jpg" {
		t.Errorf("AvatarURL = %q, want %q", info.AvatarURL, "https://example.com/avatar.jpg")
	}
}

func containsSubstr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstrHelper(s, substr))
}

func containsSubstrHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

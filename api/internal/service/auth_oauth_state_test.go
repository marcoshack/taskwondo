package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// forgeOAuthState builds a state in the current format, signed with svc's secret
// but with arbitrary fields, so tests can reach the checks behind the signature.
func forgeOAuthState(svc *AuthService, ts, nonceHex string) string {
	payload := ts + "." + nonceHex
	raw := payload + "." + hex.EncodeToString(svc.oauthStateMAC(payload))
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

const testNonceHex = "00112233445566778899aabbccddeeff"

func TestOAuthState_UniqueAndRandom(t *testing.T) {
	svc, _, _ := newTestAuthServiceWithOAuth()

	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		state, err := svc.generateOAuthState()
		if err != nil {
			t.Fatal(err)
		}
		if seen[state] {
			t.Fatalf("state %q generated twice", state)
		}
		seen[state] = true

		raw, err := base64.RawURLEncoding.DecodeString(state)
		if err != nil {
			t.Fatalf("state is not raw base64url: %v", err)
		}
		parts := strings.Split(string(raw), ".")
		if len(parts) != 3 {
			t.Fatalf("expected ts.nonce.sig, got %q", raw)
		}
		nonce, err := hex.DecodeString(parts[1])
		if err != nil || len(nonce) < 16 {
			t.Fatalf("expected at least 128-bit nonce, got %q", parts[1])
		}
	}
}

func TestOAuthState_Expired(t *testing.T) {
	svc, _, _ := newTestAuthServiceWithOAuth()
	ts := strconv.FormatInt(time.Now().Add(-OAuthStateExpiry-time.Minute).Unix(), 10)
	if err := svc.validateOAuthState(forgeOAuthState(svc, ts, testNonceHex)); err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expected expired error, got %v", err)
	}
}

func TestOAuthState_FutureTimestamp(t *testing.T) {
	svc, _, _ := newTestAuthServiceWithOAuth()
	ts := strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10)
	if err := svc.validateOAuthState(forgeOAuthState(svc, ts, testNonceHex)); err == nil {
		t.Fatal("expected error for future-dated state")
	}
}

func TestOAuthState_JustInsideExpiry(t *testing.T) {
	svc, _, _ := newTestAuthServiceWithOAuth()
	ts := strconv.FormatInt(time.Now().Add(-OAuthStateExpiry+time.Minute).Unix(), 10)
	if err := svc.validateOAuthState(forgeOAuthState(svc, ts, testNonceHex)); err != nil {
		t.Fatalf("expected valid state, got %v", err)
	}
}

func TestOAuthState_RejectsMalformed(t *testing.T) {
	svc, _, _ := newTestAuthServiceWithOAuth()
	now := strconv.FormatInt(time.Now().Unix(), 10)
	valid, _ := svc.generateOAuthState()
	raw, _ := base64.RawURLEncoding.DecodeString(valid)
	parts := strings.Split(string(raw), ".")

	// The pre-fix format: base64url(ts "." HMAC(secret, ts)) with padding.
	legacyMAC := hmac.New(sha256.New, svc.jwtSecret)
	legacyMAC.Write([]byte(now))
	legacy := base64.URLEncoding.EncodeToString([]byte(now + "." + hex.EncodeToString(legacyMAC.Sum(nil))))

	b64 := func(s string) string { return base64.RawURLEncoding.EncodeToString([]byte(s)) }

	cases := map[string]string{
		"legacy format":        legacy,
		"two parts":            b64(now + "." + parts[2]),
		"four parts":           b64(string(raw) + ".x"),
		"tampered nonce":       b64(parts[0] + "." + testNonceHex + "." + parts[2]),
		"tampered timestamp":   b64(strconv.FormatInt(time.Now().Unix()+1, 10) + "." + parts[1] + "." + parts[2]),
		"short nonce":          forgeOAuthState(svc, now, "0011"),
		"non-hex nonce":        forgeOAuthState(svc, now, strings.Repeat("zz", 16)),
		"non-hex signature":    b64(parts[0] + "." + parts[1] + ".nothex"),
		"non-numeric ts":       forgeOAuthState(svc, "soon", testNonceHex),
		"padded base64":        valid + "==",
		"too long":             strings.Repeat("A", oauthStateMaxLen+1),
		"empty":                "",
		"signature from other": b64(parts[0] + "." + parts[1] + "." + strings.Repeat("00", 32)),
	}
	for name, state := range cases {
		t.Run(name, func(t *testing.T) {
			if err := svc.validateOAuthState(state); err == nil {
				t.Fatalf("expected %s state to be rejected", name)
			}
		})
	}
}

func TestOAuthURL_ReturnsStateEmbeddedInURL(t *testing.T) {
	svc, _, _ := newTestAuthServiceWithOAuth()

	authURL, state, err := svc.OAuthURL(context.Background(), "discord")
	if err != nil {
		t.Fatal(err)
	}
	if state == "" || !strings.Contains(authURL, "state="+state) {
		t.Fatalf("expected state %q in URL %s", state, authURL)
	}
	if err := svc.validateOAuthState(state); err != nil {
		t.Fatalf("returned state does not validate: %v", err)
	}

	_, state2, _ := svc.OAuthURL(context.Background(), "discord")
	if state == state2 {
		t.Fatal("expected back-to-back states to differ")
	}
}

// newCountingDiscordServer stands in for Discord and counts token exchanges, so
// a test can prove a rejected callback never reaches the provider.
func newCountingDiscordServer(t *testing.T, svc *AuthService) *atomic.Int32 {
	t.Helper()
	var exchanges atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/oauth2/token":
			exchanges.Add(1)
			_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "tok", "token_type": "Bearer"})
		case "/api/v10/users/@me":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "42", "username": "bound", "email": "bound@example.com", "verified": true,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	dp := svc.providers["discord"].(*DiscordProvider)
	dp.httpClient = srv.Client()
	dp.baseURL = srv.URL
	return &exchanges
}

func TestOAuthCallback_RequiresBoundState(t *testing.T) {
	svc, _, _ := newTestAuthServiceWithOAuth()
	exchanges := newCountingDiscordServer(t, svc)

	state, _ := svc.generateOAuthState()
	other, _ := svc.generateOAuthState()

	for name, bound := range map[string]string{
		"missing":    "",
		"mismatched": other,
		"prefix":     state[:len(state)-1],
	} {
		t.Run(name, func(t *testing.T) {
			_, _, err := svc.OAuthCallback(context.Background(), "discord", "code", state, bound)
			if !errors.Is(err, ErrOAuthStateNotBound) {
				t.Fatalf("expected ErrOAuthStateNotBound, got %v", err)
			}
		})
	}
	if n := exchanges.Load(); n != 0 {
		t.Fatalf("expected no code exchange for unbound state, got %d", n)
	}

	// A matching binding still needs a valid signature.
	_, _, err := svc.OAuthCallback(context.Background(), "discord", "code", "forged", "forged")
	if err == nil || errors.Is(err, ErrOAuthStateNotBound) {
		t.Fatalf("expected signature failure for forged state, got %v", err)
	}

	token, user, err := svc.OAuthCallback(context.Background(), "discord", "code", state, state)
	if err != nil {
		t.Fatalf("expected success with bound state, got %v", err)
	}
	if token == "" || user.Email != "bound@example.com" {
		t.Fatalf("unexpected login result: token=%q user=%+v", token, user)
	}
	if n := exchanges.Load(); n != 1 {
		t.Fatalf("expected exactly one code exchange, got %d", n)
	}
}

package handler

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/marcoshack/taskwondo/internal/model"
	"github.com/marcoshack/taskwondo/internal/service"
)

const oauthTestSecret = "test-secret-that-is-at-least-32!"

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// fakeDiscordClient answers Discord's token and user endpoints in-process and
// counts token exchanges, so a test can tell whether a callback reached the IdP.
func fakeDiscordClient(exchanges *atomic.Int32, userID, email string) *http.Client {
	return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var body any
		switch r.URL.Path {
		case "/api/oauth2/token":
			exchanges.Add(1)
			body = map[string]string{"access_token": "mock-access-token", "token_type": "Bearer"}
		case "/api/v10/users/@me":
			body = map[string]any{"id": userID, "username": "oauthuser", "email": email, "verified": true}
		default:
			return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader("")), Header: http.Header{}}, nil
		}
		b, _ := json.Marshal(body)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(b)),
			Header:     http.Header{"Content-Type": {"application/json"}},
		}, nil
	})}
}

type oauthTestEnv struct {
	router    http.Handler
	userRepo  *mockUserRepo
	exchanges *atomic.Int32
	// cookie is the binding cookie's name for this env's base URL.
	cookie string
}

// newOAuthTestEnv wires the OAuth routes on a chi router under /api/v1, as
// main.go does, with a Discord provider backed by fakeDiscordClient.
func newOAuthTestEnv(t *testing.T, baseURL string) *oauthTestEnv {
	t.Helper()
	exchanges := &atomic.Int32{}
	userRepo := newMockUserRepo()
	discord := service.NewDiscordProvider("test-client-id", "test-client-secret",
		"http://localhost:5173/auth/discord/callback", fakeDiscordClient(exchanges, "4242", "oauth@example.com"))
	authSvc := service.NewAuthService(userRepo, newMockAPIKeyRepo(), newMockOAuthAccountRepo(),
		oauthTestSecret, time.Hour, []service.OAuthProvider{discord})
	h := NewAuthHandler(authSvc, nil, nil, nil, baseURL)

	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/auth/{provider}", h.OAuthAuth)
		r.Post("/auth/{provider}/callback", h.OAuthCallback)
	})
	return &oauthTestEnv{router: r, userRepo: userRepo, exchanges: exchanges, cookie: h.stateCookieName}
}

func findCookie(t *testing.T, resp *http.Response, name string) *http.Cookie {
	t.Helper()
	var found *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == name {
			if found != nil {
				t.Fatalf("cookie %s set more than once", name)
			}
			found = c
		}
	}
	return found
}

// startLogin calls GET /auth/discord and returns the state embedded in the
// authorization URL and the binding cookie set alongside it.
func (e *oauthTestEnv) startLogin(t *testing.T) (string, *http.Cookie) {
	t.Helper()
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/auth/discord", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(resp.Data.URL)
	if err != nil {
		t.Fatal(err)
	}
	state := u.Query().Get("state")
	if state == "" {
		t.Fatalf("no state in authorization URL %s", resp.Data.URL)
	}
	c := findCookie(t, w.Result(), e.cookie)
	if c == nil {
		t.Fatal("expected OAuth state cookie on authorization response")
	}
	return state, c
}

// callback posts {code,state} to the callback, sending cookieValue as the
// binding cookie unless it is empty.
func (e *oauthTestEnv) callback(t *testing.T, state, cookieValue string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"code": "the-code", "state": state})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/discord/callback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if cookieValue != "" {
		req.AddCookie(&http.Cookie{Name: e.cookie, Value: cookieValue})
	}
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return w
}

func assertOAuthError(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error.Code != CodeOAuthError {
		t.Fatalf("expected %s, got %q", CodeOAuthError, resp.Error.Code)
	}
	if strings.Contains(w.Body.String(), `"token"`) {
		t.Fatalf("rejected callback must not return a token: %s", w.Body.String())
	}
}

func (e *oauthTestEnv) assertStateCookieCleared(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()
	c := findCookie(t, w.Result(), e.cookie)
	if c == nil {
		t.Fatal("expected callback to clear the OAuth state cookie")
	}
	if c.MaxAge >= 0 || c.Value != "" {
		t.Fatalf("expected an expired, empty cookie, got MaxAge=%d Value=%q", c.MaxAge, c.Value)
	}
	if c.Path != oauthStateCookiePath {
		t.Fatalf("clearing cookie must use the same Path, got %q", c.Path)
	}
}

// signState signs a state the way the service does (the format is
// base64url("<unix>.<nonce hex>.<hmac hex>")) issued `age` ago, so the test can
// prove a validly signed but stale state is rejected over HTTP.
// TestOAuthCallback_SignStateMatchesService guards against this drifting.
func signState(age time.Duration) string {
	return signStateWith(oauthTestSecret, age)
}

// signStateWith signs a well-formed state with an arbitrary secret, so a test
// can send a state whose structure parses but whose HMAC does not verify.
func signStateWith(secret string, age time.Duration) string {
	ts := strconv.FormatInt(time.Now().Add(-age).Unix(), 10)
	payload := ts + ".00112233445566778899aabbccddeeff"
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte("taskwondo-oauth-state:v2:"))
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString([]byte(payload + "." + hex.EncodeToString(mac.Sum(nil))))
}

func TestOAuthAuth_SetsStateCookie(t *testing.T) {
	cases := []struct {
		baseURL string
		secure  bool
	}{
		{"http://localhost:3000", false},
		{"https://tasks.example.com", true},
		{"HTTPS://tasks.example.com/", true},
		{" https://tasks.example.com ", true},
		{"", false},
		{"://not a url", false},
	}
	for _, tc := range cases {
		wantName := "taskwondo_oauth_state"
		if tc.secure {
			wantName = "__Secure-taskwondo_oauth_state"
		}
		if got := newOAuthTestEnv(t, tc.baseURL).cookie; got != wantName {
			t.Fatalf("base URL %q: expected cookie name %q, got %q", tc.baseURL, wantName, got)
		}
	}
	for _, tc := range cases {
		t.Run(tc.baseURL, func(t *testing.T) {
			env := newOAuthTestEnv(t, tc.baseURL)

			w := httptest.NewRecorder()
			env.router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/auth/discord", nil))
			if w.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", w.Code)
			}
			if cc := w.Header().Get("Cache-Control"); cc != "no-store" {
				t.Fatalf("expected Cache-Control no-store, got %q", cc)
			}

			c := findCookie(t, w.Result(), env.cookie)
			if c == nil {
				t.Fatal("expected state cookie")
			}
			if !c.HttpOnly {
				t.Error("expected HttpOnly")
			}
			if c.SameSite != http.SameSiteLaxMode {
				t.Errorf("expected SameSite=Lax, got %v", c.SameSite)
			}
			if c.Path != "/api/v1/auth" {
				t.Errorf("expected Path=/api/v1/auth, got %q", c.Path)
			}
			// Literal on purpose: a change to the shared expiry constant must
			// fail here rather than move the cookie along with it.
			if c.MaxAge != 600 {
				t.Errorf("expected Max-Age=600, got %d", c.MaxAge)
			}
			if want := int(service.OAuthStateExpiry / time.Second); c.MaxAge != want {
				t.Errorf("expected Max-Age to match the state expiry %d, got %d", want, c.MaxAge)
			}
			if c.Secure != tc.secure {
				t.Errorf("expected Secure=%v, got %v", tc.secure, c.Secure)
			}
			if c.Domain != "" {
				t.Errorf("expected host-only cookie, got Domain=%q", c.Domain)
			}

			var resp struct {
				Data struct {
					URL string `json:"url"`
				} `json:"data"`
			}
			_ = json.Unmarshal(w.Body.Bytes(), &resp)
			u, _ := url.Parse(resp.Data.URL)
			if got := u.Query().Get("state"); got == "" || got != c.Value {
				t.Fatalf("cookie value %q must equal the URL state %q", c.Value, got)
			}
		})
	}
}

func TestOAuthAuth_NotConfiguredSetsNoCookie(t *testing.T) {
	env := newOAuthTestEnv(t, "https://tasks.example.com")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/auth/google", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	if c := findCookie(t, w.Result(), env.cookie); c != nil {
		t.Fatalf("expected no cookie for unconfigured provider, got %+v", c)
	}
}

func TestOAuthAuth_BackToBackStatesDiffer(t *testing.T) {
	env := newOAuthTestEnv(t, "http://localhost:3000")
	s1, c1 := env.startLogin(t)
	s2, c2 := env.startLogin(t)
	if s1 == s2 || c1.Value == c2.Value {
		t.Fatalf("expected distinct states, got %q twice", s1)
	}
}

func TestOAuthCallback_SucceedsWithMatchingCookie(t *testing.T) {
	env := newOAuthTestEnv(t, "https://tasks.example.com")
	state, cookie := env.startLogin(t)

	w := env.callback(t, state, cookie.Value)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			Token string `json:"token"`
			User  struct {
				Email string `json:"email"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Data.Token == "" || resp.Data.User.Email != "oauth@example.com" {
		t.Fatalf("unexpected login response: %s", w.Body.String())
	}
	if env.exchanges.Load() != 1 {
		t.Fatalf("expected one code exchange, got %d", env.exchanges.Load())
	}
	env.assertStateCookieCleared(t, w)
	if c := findCookie(t, w.Result(), env.cookie); !c.Secure || !c.HttpOnly || c.SameSite != http.SameSiteLaxMode {
		t.Fatalf("clearing cookie must keep the same attributes, got %+v", c)
	}
	if cc := w.Header().Get("Cache-Control"); cc != "no-store" {
		t.Fatalf("expected Cache-Control no-store, got %q", cc)
	}
}

func TestOAuthCallback_RejectsUnboundState(t *testing.T) {
	env := newOAuthTestEnv(t, "http://localhost:3000")

	// The attack: the state and code come from a login the attacker started in
	// their own browser; the victim's browser has no cookie, or its own one.
	attackerState, _ := env.startLogin(t)
	_, victimCookie := env.startLogin(t)

	cases := map[string]struct {
		state, cookie string
	}{
		"no cookie":                  {attackerState, ""},
		"cookie from another login":  {attackerState, victimCookie.Value},
		"cookie is a prefix":         {attackerState, attackerState[:len(attackerState)-2]},
		"forged state, cookie match": {"Zm9yZ2Vk", "Zm9yZ2Vk"},
		// Well-formed, fresh, but signed with another secret: only the HMAC
		// check can reject it.
		"wrong signature, cookie match": {
			signStateWith("another-secret-that-is-32-bytes!", 0), signStateWith("another-secret-that-is-32-bytes!", 0),
		},
		"expired state, cookie match": {
			signState(service.OAuthStateExpiry + time.Minute), signState(service.OAuthStateExpiry + time.Minute),
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			w := env.callback(t, tc.state, tc.cookie)
			assertOAuthError(t, w)
			env.assertStateCookieCleared(t, w)
		})
	}
	if n := env.exchanges.Load(); n != 0 {
		t.Fatalf("rejected callbacks must not exchange the code, got %d exchanges", n)
	}
	if len(env.userRepo.byID) != 0 {
		t.Fatalf("rejected callbacks must not create users, got %d", len(env.userRepo.byID))
	}
}

func TestOAuthCallback_ClearsCookieOnBadRequest(t *testing.T) {
	env := newOAuthTestEnv(t, "http://localhost:3000")
	_, cookie := env.startLogin(t)

	for name, body := range map[string]string{
		"invalid json":   "not json",
		"missing fields": `{"code":""}`,
	} {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/discord/callback", strings.NewReader(body))
			req.AddCookie(cookie)
			w := httptest.NewRecorder()
			env.router.ServeHTTP(w, req)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", w.Code)
			}
			env.assertStateCookieCleared(t, w)
		})
	}
}

func TestOAuthCallback_DisabledAccountClearsCookie(t *testing.T) {
	env := newOAuthTestEnv(t, "http://localhost:3000")
	state, cookie := env.startLogin(t)

	// A user with the provider's verified email exists but is disabled.
	disabled := &model.User{Email: "oauth@example.com", DisplayName: "Off", GlobalRole: model.RoleUser, IsActive: false}
	_ = env.userRepo.Create(context.Background(), disabled)

	w := env.callback(t, state, cookie.Value)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	env.assertStateCookieCleared(t, w)
}

// A fresh state from signState is accepted, so the expired case above fails on
// expiry rather than on a signature that no longer matches the service's.
func TestOAuthCallback_SignStateMatchesService(t *testing.T) {
	env := newOAuthTestEnv(t, "http://localhost:3000")
	state := signState(0)
	if w := env.callback(t, state, state); w.Code != http.StatusOK {
		t.Fatalf("expected fresh signed state to be accepted, got %d: %s", w.Code, w.Body.String())
	}
}

// A cookie tossed in next to ours (e.g. by a sibling subdomain, with a more
// specific Path that the browser sends first) must not be trusted, even when it
// matches the submitted state.
func TestOAuthCallback_RejectsDuplicateStateCookies(t *testing.T) {
	for _, baseURL := range []string{"http://localhost:3000", "https://tasks.example.com"} {
		t.Run(baseURL, func(t *testing.T) {
			env := newOAuthTestEnv(t, baseURL)
			attackerState, _ := env.startLogin(t)
			_, victimCookie := env.startLogin(t)

			for name, order := range map[string][]string{
				"attacker first": {attackerState, victimCookie.Value},
				"attacker last":  {victimCookie.Value, attackerState},
			} {
				t.Run(name, func(t *testing.T) {
					body, _ := json.Marshal(map[string]string{"code": "the-code", "state": attackerState})
					req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/discord/callback", bytes.NewReader(body))
					for _, v := range order {
						req.AddCookie(&http.Cookie{Name: env.cookie, Value: v})
					}
					w := httptest.NewRecorder()
					env.router.ServeHTTP(w, req)
					assertOAuthError(t, w)
					env.assertStateCookieCleared(t, w)
				})
			}
			if n := env.exchanges.Load(); n != 0 {
				t.Fatalf("expected no code exchange, got %d", n)
			}
		})
	}
}

// On an https deployment only the __Secure- cookie binds the state: a plain
// cookie, which an attacker on a plain-http response could have planted, is
// ignored.
func TestOAuthCallback_SecureDeploymentIgnoresUnprefixedCookie(t *testing.T) {
	env := newOAuthTestEnv(t, "https://tasks.example.com")
	state, _ := env.startLogin(t)

	body, _ := json.Marshal(map[string]string{"code": "the-code", "state": state})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/discord/callback", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "taskwondo_oauth_state", Value: state})
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)
	assertOAuthError(t, w)
	env.assertStateCookieCleared(t, w)
	if n := env.exchanges.Load(); n != 0 {
		t.Fatalf("expected no code exchange, got %d", n)
	}
}

func TestOAuthAuth_RejectsCrossSiteStart(t *testing.T) {
	env := newOAuthTestEnv(t, "https://tasks.example.com")
	for _, site := range []string{"cross-site", "same-site"} {
		t.Run(site, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/discord", nil)
			req.Header.Set("Sec-Fetch-Site", site)
			w := httptest.NewRecorder()
			env.router.ServeHTTP(w, req)
			if w.Code != http.StatusForbidden {
				t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
			}
			if c := findCookie(t, w.Result(), env.cookie); c != nil {
				t.Fatalf("cross-site start must not set the state cookie, got %+v", c)
			}
		})
	}
	for _, site := range []string{"same-origin", "none", ""} {
		t.Run("allowed "+site, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/discord", nil)
			if site != "" {
				req.Header.Set("Sec-Fetch-Site", site)
			}
			w := httptest.NewRecorder()
			env.router.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
			}
			if c := findCookie(t, w.Result(), env.cookie); c == nil {
				t.Fatal("expected state cookie")
			}
		})
	}
}

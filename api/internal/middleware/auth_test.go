package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/marcoshack/taskwondo/internal/model"
)

type mockAuthenticator struct {
	jwtInfo    *model.AuthInfo
	apiKeyInfo *model.AuthInfo
}

func (m *mockAuthenticator) ValidateJWT(_ string) (*model.AuthInfo, error) {
	if m.jwtInfo == nil {
		return nil, model.ErrUnauthorized
	}
	return m.jwtInfo, nil
}

func (m *mockAuthenticator) ValidateAPIKey(_ context.Context, _ string) (*model.AuthInfo, error) {
	if m.apiKeyInfo == nil {
		return nil, model.ErrUnauthorized
	}
	return m.apiKeyInfo, nil
}

func TestAuth_APIKeyPermission_ReadOnlyAllowsGET(t *testing.T) {
	auth := &mockAuthenticator{
		apiKeyInfo: &model.AuthInfo{
			Email:       "test@example.com",
			GlobalRole:  "user",
			Permissions: []string{"read"},
		},
	}

	handler := Auth(auth)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/default/projects", nil)
	req.Header.Set("Authorization", "Bearer twk_testkey123")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for GET with read permission, got %d", w.Code)
	}
}

func TestAuth_APIKeyPermission_ReadOnlyDeniesPOST(t *testing.T) {
	auth := &mockAuthenticator{
		apiKeyInfo: &model.AuthInfo{
			Email:       "test@example.com",
			GlobalRole:  "user",
			Permissions: []string{"read"},
		},
	}

	handler := Auth(auth)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/default/projects", nil)
	req.Header.Set("Authorization", "Bearer twk_testkey123")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for POST with read-only permission, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	errObj := resp["error"].(map[string]interface{})
	if errObj["message"] != "api key does not have sufficient permissions" {
		t.Fatalf("unexpected error message: %v", errObj["message"])
	}
}

func TestAuth_APIKeyPermission_WriteAllowsPOST(t *testing.T) {
	auth := &mockAuthenticator{
		apiKeyInfo: &model.AuthInfo{
			Email:       "test@example.com",
			GlobalRole:  "user",
			Permissions: []string{"write"},
		},
	}

	handler := Auth(auth)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/default/projects", nil)
	req.Header.Set("Authorization", "Bearer twk_testkey123")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for POST with write permission, got %d", w.Code)
	}
}

func TestAuth_APIKeyPermission_EmptyAllowsAll(t *testing.T) {
	auth := &mockAuthenticator{
		apiKeyInfo: &model.AuthInfo{
			Email:       "test@example.com",
			GlobalRole:  "user",
			Permissions: []string{},
		},
	}

	handler := Auth(auth)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodDelete} {
		req := httptest.NewRequest(method, "/api/v1/default/projects", nil)
		req.Header.Set("Authorization", "Bearer twk_testkey123")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 for %s with empty permissions, got %d", method, w.Code)
		}
	}
}

func TestAuth_JWTNoPermissionCheck(t *testing.T) {
	auth := &mockAuthenticator{
		jwtInfo: &model.AuthInfo{
			Email:      "test@example.com",
			GlobalRole: "user",
		},
	}

	handler := Auth(auth)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/default/projects", nil)
	req.Header.Set("Authorization", "Bearer jwt-token-here")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for JWT (no permission restrictions), got %d", w.Code)
	}
}

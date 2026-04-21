package handler

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func TestParseUUIDParam_Success(t *testing.T) {
	want := uuid.New()
	r := httptest.NewRequest("GET", "/x/"+want.String(), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", want.String())
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	got, ok := parseUUIDParam(w, r, "id", "invalid id")
	if !ok {
		t.Fatalf("expected ok=true, got body=%s", w.Body.String())
	}
	if got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
	if w.Code != 200 {
		t.Fatalf("expected no write; got status %d", w.Code)
	}
}

func TestParseUUIDParam_InvalidReturnsError(t *testing.T) {
	r := httptest.NewRequest("GET", "/x/nope", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "not-a-uuid")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	_, ok := parseUUIDParam(w, r, "id", "invalid id")
	if ok {
		t.Fatal("expected ok=false")
	}
	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var resp map[string]map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["error"]["code"] != CodeValidationError {
		t.Fatalf("expected code %s, got %s", CodeValidationError, resp["error"]["code"])
	}
	if resp["error"]["message"] != "invalid id" {
		t.Fatalf("expected message 'invalid id', got %q", resp["error"]["message"])
	}
}

func TestUnmarshalField_WrongTypeReturnsError(t *testing.T) {
	// Previously, Update handlers silently dropped fields that failed Unmarshal;
	// the helper now rejects with a validation error so bad payloads are visible.
	raw := json.RawMessage(`[1,2,3]`)
	w := httptest.NewRecorder()
	_, ok := unmarshalField[string](w, raw, "title")
	if ok {
		t.Fatal("expected ok=false for type mismatch")
	}
	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var resp map[string]map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["error"]["code"] != CodeValidationError {
		t.Fatalf("expected code %s, got %s", CodeValidationError, resp["error"]["code"])
	}
}

func TestUnmarshalField_Success(t *testing.T) {
	raw := json.RawMessage(`"hello"`)
	w := httptest.NewRecorder()
	got, ok := unmarshalField[string](w, raw, "title")
	if !ok {
		t.Fatalf("expected ok=true, got body=%s", w.Body.String())
	}
	if got != "hello" {
		t.Fatalf("got %q, want %q", got, "hello")
	}
}

func TestUnmarshalNullableUUID_Null(t *testing.T) {
	raw := json.RawMessage(`null`)
	w := httptest.NewRecorder()
	id, cleared, ok := unmarshalNullableUUID(w, raw, "assignee_id")
	if !ok || !cleared {
		t.Fatalf("expected cleared=true ok=true; got cleared=%v ok=%v", cleared, ok)
	}
	if id != uuid.Nil {
		t.Fatalf("expected uuid.Nil, got %s", id)
	}
}

func TestUnmarshalNullableUUID_ValidID(t *testing.T) {
	want := uuid.New()
	raw := json.RawMessage(`"` + want.String() + `"`)
	w := httptest.NewRecorder()
	got, cleared, ok := unmarshalNullableUUID(w, raw, "assignee_id")
	if !ok || cleared {
		t.Fatalf("expected ok=true cleared=false; got ok=%v cleared=%v", ok, cleared)
	}
	if got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestUnmarshalNullableUUID_InvalidID(t *testing.T) {
	raw := json.RawMessage(`"not-a-uuid"`)
	w := httptest.NewRecorder()
	_, _, ok := unmarshalNullableUUID(w, raw, "assignee_id")
	if ok {
		t.Fatal("expected ok=false")
	}
	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestUnmarshalNullableDate_Null(t *testing.T) {
	raw := json.RawMessage(`null`)
	w := httptest.NewRecorder()
	_, cleared, ok := unmarshalNullableDate(w, raw, "due_date")
	if !ok || !cleared {
		t.Fatalf("expected cleared=true ok=true; got cleared=%v ok=%v", cleared, ok)
	}
}

func TestUnmarshalNullableDate_Valid(t *testing.T) {
	raw := json.RawMessage(`"2026-04-21"`)
	w := httptest.NewRecorder()
	got, cleared, ok := unmarshalNullableDate(w, raw, "due_date")
	if !ok || cleared {
		t.Fatalf("expected ok=true cleared=false; got ok=%v cleared=%v", ok, cleared)
	}
	if got.Year() != 2026 || got.Month() != 4 || got.Day() != 21 {
		t.Fatalf("unexpected date: %v", got)
	}
}

func TestUnmarshalNullableDate_BadFormat(t *testing.T) {
	raw := json.RawMessage(`"tomorrow"`)
	w := httptest.NewRecorder()
	_, _, ok := unmarshalNullableDate(w, raw, "due_date")
	if ok {
		t.Fatal("expected ok=false")
	}
	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

package i18n

import "testing"

func TestT_English(t *testing.T) {
	got := T("en", "email.assignment.cta")
	if got != "View Work Item" {
		t.Errorf("expected 'View Work Item', got %q", got)
	}
}

func TestT_Substitution(t *testing.T) {
	got := T("en", "email.assignment.subject",
		"projectKey", "TP",
		"itemNumber", "42",
		"title", "Fix bug",
	)
	want := "[TP] Work item #42 assigned to you: Fix bug"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestT_FallbackToEnglish(t *testing.T) {
	// Use a language that doesn't exist — should fall back to English
	got := T("xx", "email.assignment.cta")
	if got != "View Work Item" {
		t.Errorf("expected English fallback, got %q", got)
	}
}

func TestT_MissingKeyReturnsKey(t *testing.T) {
	got := T("en", "nonexistent.key")
	if got != "nonexistent.key" {
		t.Errorf("expected key returned as-is, got %q", got)
	}
}

func TestT_Portuguese(t *testing.T) {
	got := T("pt", "email.assignment.cta")
	if got != "Ver Item de Trabalho" {
		t.Errorf("expected Portuguese CTA, got %q", got)
	}
}

func TestT_Japanese(t *testing.T) {
	got := T("ja", "email.member_added.cta")
	want := "プロジェクトを表示"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestT_AllLanguagesLoaded(t *testing.T) {
	expected := []string{"en", "pt", "es", "fr", "de", "ja", "ko", "zh", "ar"}
	for _, lang := range expected {
		if _, ok := translations[lang]; !ok {
			t.Errorf("language %q not loaded", lang)
		}
	}
}

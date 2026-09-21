package i18n

import (
	"regexp"
	"sort"
	"strings"
	"testing"
)

var placeholderRe = regexp.MustCompile(`\{\{(\w+)\}\}`)

func TestTranslations_KeyParity(t *testing.T) {
	en := translations["en"]
	if len(en) == 0 {
		t.Fatal("no English translations loaded")
	}
	for lang, m := range translations {
		if len(m) != len(en) {
			t.Errorf("%s: has %d keys, en has %d", lang, len(m), len(en))
		}
		for key := range en {
			if _, ok := m[key]; !ok {
				t.Errorf("%s: missing key %q", lang, key)
			}
		}
		for key := range m {
			if _, ok := en[key]; !ok {
				t.Errorf("%s: extra key %q", lang, key)
			}
		}
	}
}

func TestTranslations_PlaceholderConsistency(t *testing.T) {
	for key, enVal := range translations["en"] {
		want := placeholders(enVal)
		for lang, m := range translations {
			if lang == "en" {
				continue
			}
			got := placeholders(m[key])
			if strings.Join(got, ",") != strings.Join(want, ",") {
				t.Errorf("%s: key %q has placeholders %v, en has %v", lang, key, got, want)
			}
		}
	}
}

func placeholders(s string) []string {
	var out []string
	seen := map[string]bool{}
	for _, m := range placeholderRe.FindAllStringSubmatch(s, -1) {
		if !seen[m[1]] {
			seen[m[1]] = true
			out = append(out, m[1])
		}
	}
	sort.Strings(out)
	return out
}

func TestNegotiate(t *testing.T) {
	cases := map[string]string{
		"zh-CN,zh;q=0.9,en;q=0.8":    "zh",
		"en-US,en;q=0.9":             "en",
		"pt-BR":                      "pt",
		"fr":                         "fr",
		"de-DE,de;q=0.9":             "de",
		"nl,es;q=0.8":                "es",
		"sw":                         "en",
		"":                           "en",
		"ZH":                         "zh",
		"ko-KR,ko;q=0.9":             "ko",
		"ar,en;q=0.9":                "ar",
		"ja":                         "ja",
		"en-GB,en;q=0.9,zh-CN;q=0.8": "en",
	}
	for header, want := range cases {
		if got := Negotiate(header); got != want {
			t.Errorf("Negotiate(%q) = %q, want %q", header, got, want)
		}
	}
}

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

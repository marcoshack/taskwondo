package repository

import (
	"reflect"
	"testing"
)

func TestContainsCJK(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"empty", "", false},
		{"ascii", "login crash", false},
		{"han", "登录崩溃", true},
		{"mixed ascii and han", "fix 登录", true},
		{"hiragana", "ページ", true},
		{"katakana", "バグ", true},
		{"hangul", "오류", true},
		{"punctuation only", "！？。**", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := containsCJK(tt.input); got != tt.want {
				t.Errorf("containsCJK(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestLikePattern(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain", "登录", "%登录%"},
		{"escapes percent", "100%", "%100\\%%"},
		{"escapes underscore", "a_b", "%a\\_b%"},
		{"escapes backslash", `a\b`, `%a\\b%`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := likePattern(tt.input); got != tt.want {
				t.Errorf("likePattern(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSearchFilterCondition(t *testing.T) {
	t.Run("latin query stays tsvector-only", func(t *testing.T) {
		cond, args := searchFilterCondition("search_vector", []string{"title"}, "login")
		want := "(search_vector @@ plainto_tsquery('english', ?) OR search_vector @@ plainto_tsquery('simple', ?))"
		if cond != want {
			t.Errorf("cond = %q, want %q", cond, want)
		}
		if !reflect.DeepEqual(args, []interface{}{"login", "login"}) {
			t.Errorf("args = %v", args)
		}
	})

	t.Run("cjk query adds ILIKE fallback per column", func(t *testing.T) {
		cond, args := searchFilterCondition("w.search_vector", []string{"w.title", "coalesce(w.description, '')"}, "登录")
		want := "(w.search_vector @@ plainto_tsquery('english', ?) OR w.search_vector @@ plainto_tsquery('simple', ?)" +
			" OR w.title ILIKE ? OR coalesce(w.description, '') ILIKE ?)"
		if cond != want {
			t.Errorf("cond = %q, want %q", cond, want)
		}
		if len(args) != 4 {
			t.Fatalf("len(args) = %d, want 4", len(args))
		}
		for _, a := range args[2:] {
			if a != "%登录%" {
				t.Errorf("ILIKE arg = %v, want %%登录%%", a)
			}
		}
	})

	t.Run("placeholder count matches arg count", func(t *testing.T) {
		cond, args := searchFilterCondition("search_vector", []string{"title", "description", "display_id"}, "한글")
		n := 0
		for _, r := range cond {
			if r == '?' {
				n++
			}
		}
		if n != len(args) {
			t.Errorf("cond has %d placeholders but %d args", n, len(args))
		}
	})
}

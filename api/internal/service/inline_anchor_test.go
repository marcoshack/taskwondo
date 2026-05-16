package service

import (
	"strings"
	"testing"
)

func TestNormalizeContent(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"strips trailing whitespace", "line one  \nline two\t\n", "line one\nline two"},
		{"normalizes CRLF", "a\r\nb\r\nc", "a\nb\nc"},
		{"drops trailing blank lines", "a\nb\n\n\n", "a\nb"},
		{"preserves internal blanks", "a\n\nb", "a\n\nb"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeContent(tt.in)
			if got != tt.want {
				t.Errorf("NormalizeContent(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestHashContent_StableForEquivalent(t *testing.T) {
	a := HashContent("hello\nworld")
	b := HashContent("hello\nworld\n")
	c := HashContent("hello  \nworld\n\n")
	if a != b || b != c {
		t.Fatalf("expected normalized hashes to match: %q %q %q", a, b, c)
	}
	d := HashContent("hello\nworld!")
	if d == a {
		t.Fatal("different content must hash differently")
	}
}

func TestFindAnchor_ExactMatch(t *testing.T) {
	content := strings.Join([]string{
		"intro paragraph",
		"",
		"## Section",
		"some details",
		"more details",
	}, "\n")

	got := FindAnchor(content, "## Section\nsome details")
	if !got.Found {
		t.Fatal("expected anchor to be found")
	}
	if got.StartLine != 3 || got.EndLine != 4 {
		t.Errorf("expected lines 3-4, got %d-%d", got.StartLine, got.EndLine)
	}
}

func TestFindAnchor_ReanchorsWhenContentMovesDown(t *testing.T) {
	original := "alpha\nbeta\ngamma"
	updated := "intro line\n\n" + original

	got := FindAnchor(updated, "beta")
	if !got.Found {
		t.Fatal("expected anchor to be found after content moved")
	}
	// "beta" is now the 4th line (intro, blank, alpha, beta)
	if got.StartLine != 4 || got.EndLine != 4 {
		t.Errorf("expected line 4, got %d-%d", got.StartLine, got.EndLine)
	}
}

func TestFindAnchor_OrphanedSnippet(t *testing.T) {
	content := "completely\nunrelated\ncontent"
	got := FindAnchor(content, "## Section\nvanished snippet")
	if got.Found {
		t.Fatal("expected orphaned snippet to NOT be found")
	}
}

func TestFindAnchor_FuzzyMatch(t *testing.T) {
	// Original snippet: 4 lines. New content has 3 of the 4 lines unchanged,
	// 1 line slightly edited. Score = 3/4 = 75% > 60% threshold.
	original := strings.Join([]string{
		"This is the first line",
		"This is the second line",
		"This is the third line",
		"This is the fourth line",
	}, "\n")
	updated := strings.Join([]string{
		"This is the first line",
		"This is the second line (edited)",
		"This is the third line",
		"This is the fourth line",
	}, "\n")

	got := FindAnchor(updated, original)
	if !got.Found {
		t.Fatal("expected fuzzy match to succeed")
	}
	if got.StartLine != 1 || got.EndLine != 4 {
		t.Errorf("expected lines 1-4, got %d-%d", got.StartLine, got.EndLine)
	}
}

func TestFindAnchor_FuzzyBelowThreshold(t *testing.T) {
	// Only 1 of 4 lines match — below the 60% threshold.
	original := strings.Join([]string{
		"alpha",
		"bravo",
		"charlie",
		"delta",
	}, "\n")
	updated := strings.Join([]string{
		"x",
		"y",
		"z",
		"delta",
	}, "\n")

	got := FindAnchor(updated, original)
	if got.Found {
		t.Fatal("expected match to fail below threshold")
	}
}

func TestFindAnchor_EmptySnippet(t *testing.T) {
	got := FindAnchor("anything", "")
	if got.Found {
		t.Fatal("expected empty snippet to never match")
	}
	got = FindAnchor("anything", "   \n\t\n")
	if got.Found {
		t.Fatal("expected blank snippet to never match")
	}
}

func TestFindAnchor_SnippetLargerThanContent(t *testing.T) {
	got := FindAnchor("short", "line1\nline2\nline3")
	if got.Found {
		t.Fatal("expected match to fail when snippet exceeds content size")
	}
}

func TestFindAnchor_AmbiguousExactFallsToFuzzy(t *testing.T) {
	// Two identical paragraphs. Exact match is ambiguous, fuzzy will pick
	// the highest-scoring one — which is still ambiguous (tie), so we
	// expect no match.
	content := "alpha\nbeta\n\nalpha\nbeta"
	got := FindAnchor(content, "alpha\nbeta")
	if got.Found {
		t.Fatalf("expected ambiguous match to be rejected, got %+v", got)
	}
}

func TestExtractSnippet(t *testing.T) {
	content := "a\nb\nc\nd\ne"
	tests := []struct {
		name             string
		start, end       int
		want             string
	}{
		{"single line", 2, 2, "b"},
		{"range", 2, 4, "b\nc\nd"},
		{"out of range start", 99, 99, ""},
		{"clipped end", 4, 99, "d\ne"},
		{"invalid range", 3, 1, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractSnippet(content, tt.start, tt.end)
			if got != tt.want {
				t.Errorf("ExtractSnippet(_, %d, %d) = %q, want %q", tt.start, tt.end, got, tt.want)
			}
		})
	}
}

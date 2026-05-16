package service

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// HashContent computes the canonical sha256 hash used by both description
// revisions and inline-comment snippets. It hashes the normalized form
// (NormalizeContent) so cosmetic whitespace differences don't show up as
// content changes.
func HashContent(content string) string {
	h := sha256.Sum256([]byte(NormalizeContent(content)))
	return hex.EncodeToString(h[:])
}

// HashSnippet hashes the raw snippet text without normalization. This is
// what the re-anchor pass compares against per-line hashes of the new
// content. We intentionally do not normalize here so leading/trailing
// whitespace within a snippet stays significant for fuzzy matching.
func HashSnippet(snippet string) string {
	h := sha256.Sum256([]byte(snippet))
	return hex.EncodeToString(h[:])
}

// NormalizeContent strips trailing whitespace from each line and collapses
// trailing blank lines. Two descriptions that differ only by trailing
// whitespace produce the same normalized form, and therefore the same hash.
func NormalizeContent(content string) string {
	// Normalize line endings.
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}

	// Drop trailing blank lines.
	end := len(lines)
	for end > 0 && lines[end-1] == "" {
		end--
	}
	return strings.Join(lines[:end], "\n")
}

// AnchorMatch is the result of trying to re-anchor a comment against new
// content. Found = true means StartLine / EndLine / Snippet are valid;
// false means the snippet could not be located.
type AnchorMatch struct {
	Found     bool
	StartLine int
	EndLine   int
	Snippet   string
}

// FindAnchor searches `newContent` for an occurrence of `originalSnippet`.
// It returns the new line range (1-based, inclusive) when found.
//
// Strategy:
//  1. Exact match — try to locate the snippet's lines verbatim. If a unique
//     match is found, use it.
//  2. Fuzzy match — if no unique exact match, score every candidate window
//     against the snippet using a per-line normalized comparison. The
//     highest-scoring window above the threshold wins (must beat the
//     runner-up by at least one line).
//
// A snippet's "lines" are obtained by splitting on '\n' after normalizing
// (NormalizeContent). Trailing/leading blank lines inside the snippet are
// kept; only line-end whitespace is stripped.
//
// The threshold for fuzzy matching is 60% — a window has to share at least
// 60% of its lines with the snippet to be considered a candidate.
func FindAnchor(newContent string, originalSnippet string) AnchorMatch {
	if strings.TrimSpace(originalSnippet) == "" {
		return AnchorMatch{}
	}

	newLines := strings.Split(NormalizeContent(newContent), "\n")
	snippetLines := normalizeLines(strings.Split(originalSnippet, "\n"))

	if len(snippetLines) == 0 {
		return AnchorMatch{}
	}

	if len(snippetLines) > len(newLines) {
		return AnchorMatch{}
	}

	// 1. Exact match.
	matches := findExactWindows(newLines, snippetLines)
	if len(matches) == 1 {
		start := matches[0]
		end := start + len(snippetLines) - 1
		return AnchorMatch{
			Found:     true,
			StartLine: start + 1,
			EndLine:   end + 1,
			Snippet:   strings.Join(newLines[start:end+1], "\n"),
		}
	}
	if len(matches) > 1 {
		// Ambiguous exact match — fall through to fuzzy, which uses scoring
		// and ties broken by position.
	}

	// 2. Fuzzy match.
	bestScore := 0
	bestStart := -1
	secondBest := 0
	windowSize := len(snippetLines)

	for i := 0; i+windowSize <= len(newLines); i++ {
		score := 0
		for j := range windowSize {
			if normalizeLine(newLines[i+j]) == normalizeLine(snippetLines[j]) {
				score++
			}
		}
		if score > bestScore {
			secondBest = bestScore
			bestScore = score
			bestStart = i
		} else if score > secondBest {
			secondBest = score
		}
	}

	threshold := max((windowSize*6)/10, 1)
	// Require the best to beat the runner-up — otherwise we have an
	// ambiguous fuzzy match and we treat the comment as outdated.
	if bestStart >= 0 && bestScore >= threshold && bestScore > secondBest {
		end := bestStart + windowSize - 1
		return AnchorMatch{
			Found:     true,
			StartLine: bestStart + 1,
			EndLine:   end + 1,
			Snippet:   strings.Join(newLines[bestStart:end+1], "\n"),
		}
	}

	return AnchorMatch{}
}

// findExactWindows returns the starting line indices (0-based) where the
// snippet appears verbatim in newLines.
func findExactWindows(newLines, snippetLines []string) []int {
	var matches []int
	if len(snippetLines) == 0 {
		return matches
	}
	first := snippetLines[0]
	for i := 0; i+len(snippetLines) <= len(newLines); i++ {
		if newLines[i] != first {
			continue
		}
		ok := true
		for j := 1; j < len(snippetLines); j++ {
			if newLines[i+j] != snippetLines[j] {
				ok = false
				break
			}
		}
		if ok {
			matches = append(matches, i)
		}
	}
	return matches
}

func normalizeLine(s string) string {
	return strings.TrimRight(s, " \t")
}

func normalizeLines(lines []string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = normalizeLine(l)
	}
	return out
}

// ExtractSnippet returns the text from `content` covering the inclusive
// 1-based line range [startLine, endLine]. Returns "" when the range is
// empty or out of bounds.
func ExtractSnippet(content string, startLine, endLine int) string {
	if startLine < 1 || endLine < startLine {
		return ""
	}
	lines := strings.Split(NormalizeContent(content), "\n")
	if startLine > len(lines) {
		return ""
	}
	if endLine > len(lines) {
		endLine = len(lines)
	}
	return strings.Join(lines[startLine-1:endLine], "\n")
}

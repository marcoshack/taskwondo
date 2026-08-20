package main

import "testing"

func TestCommentVisibilityPublicByDefault(t *testing.T) {
	t.Parallel()
	if got := commentVisibility(false); got != "public" {
		t.Fatalf("commentVisibility(false) = %q, want public", got)
	}
}

func TestCommentVisibilityInternalWhenRequested(t *testing.T) {
	t.Parallel()
	if got := commentVisibility(true); got != "internal" {
		t.Fatalf("commentVisibility(true) = %q, want internal", got)
	}
}

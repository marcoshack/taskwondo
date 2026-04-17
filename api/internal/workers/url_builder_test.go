package workers

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
)

// stubNamespaceResolver is a test implementation of namespaceResolver backed by
// an in-memory map from project ID to namespace slug.
type stubNamespaceResolver struct {
	slugs map[uuid.UUID]string
	err   error
}

func (s *stubNamespaceResolver) ResolveNamespacesByIDs(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]model.ProjectNamespaceInfo, error) {
	if s.err != nil {
		return nil, s.err
	}
	out := make(map[uuid.UUID]model.ProjectNamespaceInfo, len(ids))
	for _, id := range ids {
		if slug, ok := s.slugs[id]; ok {
			out[id] = model.ProjectNamespaceInfo{NamespaceSlug: slug}
		}
	}
	return out, nil
}

// newTestURLBuilder returns a URLBuilder configured for the example base URL
// with a nil resolver. Links resolve to the default namespace segment ("d").
func newTestURLBuilder() *URLBuilder {
	return NewURLBuilder("https://example.com", nil)
}

// newTestURLBuilderWith returns a URLBuilder backed by a resolver that knows
// the given project-to-namespace-slug mapping.
func newTestURLBuilderWith(slugs map[uuid.UUID]string) *URLBuilder {
	return NewURLBuilder("https://example.com", &stubNamespaceResolver{slugs: slugs})
}

func TestURLBuilder_WorkItem_DefaultNamespaceFallback(t *testing.T) {
	b := newTestURLBuilder()
	projectID := uuid.New()

	got := b.WorkItem(context.Background(), projectID, "TP", 42)
	want := "https://example.com/d/projects/TP/items/42"
	if got != want {
		t.Errorf("WorkItem = %q, want %q", got, want)
	}
}

func TestURLBuilder_WorkItem_DefaultSlug(t *testing.T) {
	projectID := uuid.New()
	b := newTestURLBuilderWith(map[uuid.UUID]string{projectID: "default"})

	got := b.WorkItem(context.Background(), projectID, "TP", 42)
	want := "https://example.com/d/projects/TP/items/42"
	if got != want {
		t.Errorf("WorkItem = %q, want %q", got, want)
	}
}

func TestURLBuilder_WorkItem_NonDefaultNamespace(t *testing.T) {
	projectID := uuid.New()
	b := newTestURLBuilderWith(map[uuid.UUID]string{projectID: "mhack"})

	got := b.WorkItem(context.Background(), projectID, "RALLY", 7)
	want := "https://example.com/mhack/projects/RALLY/items/7"
	if got != want {
		t.Errorf("WorkItem = %q, want %q", got, want)
	}
}

func TestURLBuilder_Project_NonDefaultNamespace(t *testing.T) {
	projectID := uuid.New()
	b := newTestURLBuilderWith(map[uuid.UUID]string{projectID: "acme"})

	got := b.Project(context.Background(), projectID, "INFRA")
	want := "https://example.com/acme/projects/INFRA"
	if got != want {
		t.Errorf("Project = %q, want %q", got, want)
	}
}

func TestURLBuilder_OncallTab_NonDefaultNamespace(t *testing.T) {
	projectID := uuid.New()
	teamID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	b := newTestURLBuilderWith(map[uuid.UUID]string{projectID: "mhack"})

	got := b.OncallTab(context.Background(), projectID, "RALLY", teamID)
	want := "https://example.com/mhack/projects/RALLY/teams/11111111-1111-1111-1111-111111111111?tab=oncall"
	if got != want {
		t.Errorf("OncallTab = %q, want %q", got, want)
	}
}

func TestURLBuilder_Invite_IsNamespaceAgnostic(t *testing.T) {
	b := newTestURLBuilder()
	got := b.Invite("abc123")
	want := "https://example.com/invite/abc123"
	if got != want {
		t.Errorf("Invite = %q, want %q", got, want)
	}
}

func TestURLBuilder_ResolverErrorFallsBackToDefault(t *testing.T) {
	projectID := uuid.New()
	b := NewURLBuilder("https://example.com", &stubNamespaceResolver{err: context.Canceled})

	got := b.WorkItem(context.Background(), projectID, "TP", 1)
	want := "https://example.com/d/projects/TP/items/1"
	if got != want {
		t.Errorf("WorkItem on resolver error = %q, want %q (default fallback)", got, want)
	}
}

func TestURLBuilder_UnknownProjectFallsBackToDefault(t *testing.T) {
	projectID := uuid.New()
	otherID := uuid.New()
	b := newTestURLBuilderWith(map[uuid.UUID]string{otherID: "mhack"})

	got := b.WorkItem(context.Background(), projectID, "TP", 1)
	want := "https://example.com/d/projects/TP/items/1"
	if got != want {
		t.Errorf("WorkItem for unknown project = %q, want %q", got, want)
	}
}

func TestToURLSegment(t *testing.T) {
	tests := []struct {
		slug string
		want string
	}{
		{"default", "d"},
		{"mhack", "mhack"},
		{"acme", "acme"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := toURLSegment(tt.slug); got != tt.want {
			t.Errorf("toURLSegment(%q) = %q, want %q", tt.slug, got, tt.want)
		}
	}
}

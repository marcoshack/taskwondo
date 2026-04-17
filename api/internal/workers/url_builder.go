package workers

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
)

// namespaceResolver looks up the namespace slug for a given project ID.
// Implemented by *repository.ProjectRepository via ResolveNamespacesByIDs.
type namespaceResolver interface {
	ResolveNamespacesByIDs(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID]model.ProjectNamespaceInfo, error)
}

// URLBuilder is the single source of truth for constructing absolute URLs to
// Taskwondo resources in notifications and emails. It resolves the correct
// namespace URL segment for each project so that links point to the right
// tenant (e.g. /mhack/projects/... rather than /d/projects/...).
type URLBuilder struct {
	baseURL  string
	resolver namespaceResolver
}

// NewURLBuilder constructs a URLBuilder. If resolver is nil, URLs fall back to
// the default namespace segment ("d"), preserving legacy behaviour.
func NewURLBuilder(baseURL string, resolver namespaceResolver) *URLBuilder {
	return &URLBuilder{baseURL: baseURL, resolver: resolver}
}

// WorkItem returns the absolute URL to a work item detail page.
func (b *URLBuilder) WorkItem(ctx context.Context, projectID uuid.UUID, projectKey string, itemNumber int) string {
	return fmt.Sprintf("%s/%s/projects/%s/items/%d", b.baseURL, b.segmentFor(ctx, projectID), projectKey, itemNumber)
}

// Project returns the absolute URL to a project's main page.
func (b *URLBuilder) Project(ctx context.Context, projectID uuid.UUID, projectKey string) string {
	return fmt.Sprintf("%s/%s/projects/%s", b.baseURL, b.segmentFor(ctx, projectID), projectKey)
}

// OncallTab returns the absolute URL to a team's on-call tab.
func (b *URLBuilder) OncallTab(ctx context.Context, projectID uuid.UUID, projectKey string, teamID uuid.UUID) string {
	return fmt.Sprintf("%s/%s/projects/%s/teams/%s?tab=oncall", b.baseURL, b.segmentFor(ctx, projectID), projectKey, teamID)
}

// Invite returns the absolute URL to the invite acceptance page (namespace-agnostic).
func (b *URLBuilder) Invite(inviteCode string) string {
	return fmt.Sprintf("%s/invite/%s", b.baseURL, inviteCode)
}

// segmentFor resolves the namespace URL segment for a project, defaulting to
// "d" on lookup failure so notifications still work if the namespace cannot be
// resolved.
func (b *URLBuilder) segmentFor(ctx context.Context, projectID uuid.UUID) string {
	if b.resolver == nil || projectID == uuid.Nil {
		return toURLSegment(model.DefaultNamespaceSlug)
	}
	info, err := b.resolver.ResolveNamespacesByIDs(ctx, []uuid.UUID{projectID})
	if err != nil {
		return toURLSegment(model.DefaultNamespaceSlug)
	}
	entry, ok := info[projectID]
	if !ok {
		return toURLSegment(model.DefaultNamespaceSlug)
	}
	return toURLSegment(entry.NamespaceSlug)
}

// toURLSegment maps a namespace slug to its URL segment. This mirrors the
// frontend helper in web/src/hooks/useNamespacePath.ts: the default namespace
// is rendered as "d", all other slugs are used verbatim.
func toURLSegment(slug string) string {
	if slug == model.DefaultNamespaceSlug {
		return "d"
	}
	return slug
}

package workers

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
	"github.com/marcoshack/taskwondo/internal/weburl"
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
	return weburl.WorkItem(b.baseURL, b.slugFor(ctx, projectID), projectKey, itemNumber)
}

// Project returns the absolute URL to a project's main page.
func (b *URLBuilder) Project(ctx context.Context, projectID uuid.UUID, projectKey string) string {
	return weburl.Project(b.baseURL, b.slugFor(ctx, projectID), projectKey)
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
// the default namespace segment on lookup failure so notifications still work
// if the namespace cannot be resolved.
func (b *URLBuilder) segmentFor(ctx context.Context, projectID uuid.UUID) string {
	return weburl.Segment(b.slugFor(ctx, projectID))
}

// slugFor resolves the namespace slug for a project, defaulting to the default
// namespace slug on lookup failure so notifications still work if the namespace
// cannot be resolved.
func (b *URLBuilder) slugFor(ctx context.Context, projectID uuid.UUID) string {
	if b.resolver == nil || projectID == uuid.Nil {
		return model.DefaultNamespaceSlug
	}
	info, err := b.resolver.ResolveNamespacesByIDs(ctx, []uuid.UUID{projectID})
	if err != nil {
		return model.DefaultNamespaceSlug
	}
	entry, ok := info[projectID]
	if !ok {
		return model.DefaultNamespaceSlug
	}
	return entry.NamespaceSlug
}

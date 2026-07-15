// Package weburl is the single source of truth for constructing absolute URLs
// to Taskwondo web resources (work items, projects, ...). Both API responses
// and background notifications build links through here so that no client ever
// has to infer the URL format.
package weburl

import (
	"fmt"

	"github.com/marcoshack/taskwondo/internal/model"
)

// Segment maps a namespace slug to its URL path segment, mirroring the frontend
// helper in web/src/hooks/useNamespacePath.ts: the default namespace is rendered
// as "d", all other slugs are used verbatim. An empty slug is treated as the
// default namespace so links remain valid when the slug cannot be resolved.
func Segment(namespaceSlug string) string {
	if namespaceSlug == "" || namespaceSlug == model.DefaultNamespaceSlug {
		return "d"
	}
	return namespaceSlug
}

// WorkItem returns the absolute URL to a work item detail page for the given
// namespace slug, project key, and item number. baseURL is the public web base
// (e.g. https://taskwondo.org) with no trailing slash.
func WorkItem(baseURL, namespaceSlug, projectKey string, itemNumber int) string {
	return fmt.Sprintf("%s/%s/projects/%s/items/%d", baseURL, Segment(namespaceSlug), projectKey, itemNumber)
}

// Project returns the absolute URL to a project's main page.
func Project(baseURL, namespaceSlug, projectKey string) string {
	return fmt.Sprintf("%s/%s/projects/%s", baseURL, Segment(namespaceSlug), projectKey)
}

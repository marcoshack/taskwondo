package middleware

import (
	"testing"

	"github.com/marcoshack/taskwondo/internal/model"
)

func TestResourceFromPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"metrics root", "/metrics", model.ResourceMetrics},
		{"metrics with suffix", "/metrics/extra", model.ResourceMetrics},
		{"items list", "/api/v1/acme/projects/TF/items", model.ResourceItems},
		{"items detail", "/api/v1/acme/projects/TF/items/42", model.ResourceItems},
		{"comments nested", "/api/v1/acme/projects/TF/items/42/comments", model.ResourceItems},
		{"projects list (not items)", "/api/v1/acme/projects", ""},
		{"teams (not items)", "/api/v1/acme/projects/TF/teams", ""},
		// Regression: a project key literally called "items" previously matched
		// as ResourceItems because the old implementation scanned for any
		// segment equal to "items".
		{"project key named items", "/api/v1/acme/projects/items/teams", ""},
		{"namespace slug named items", "/api/v1/items/projects/TF/teams", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resourceFromPath(tt.path); got != tt.want {
				t.Fatalf("resourceFromPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

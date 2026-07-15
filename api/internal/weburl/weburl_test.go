package weburl

import "testing"

func TestSegment(t *testing.T) {
	tests := []struct {
		slug string
		want string
	}{
		{"default", "d"},
		{"", "d"},
		{"mhack", "mhack"},
		{"acme", "acme"},
	}
	for _, tt := range tests {
		if got := Segment(tt.slug); got != tt.want {
			t.Errorf("Segment(%q) = %q, want %q", tt.slug, got, tt.want)
		}
	}
}

func TestWorkItem(t *testing.T) {
	tests := []struct {
		name          string
		baseURL       string
		namespaceSlug string
		projectKey    string
		itemNumber    int
		want          string
	}{
		{
			name:          "default namespace maps to d segment",
			baseURL:       "https://taskwondo.org",
			namespaceSlug: "default",
			projectKey:    "TF",
			itemNumber:    389,
			want:          "https://taskwondo.org/d/projects/TF/items/389",
		},
		{
			name:          "non-default namespace uses slug verbatim",
			baseURL:       "https://taskwondo.org",
			namespaceSlug: "acme",
			projectKey:    "TF",
			itemNumber:    141,
			want:          "https://taskwondo.org/acme/projects/TF/items/141",
		},
		{
			name:          "empty slug falls back to default segment",
			baseURL:       "http://localhost:3000",
			namespaceSlug: "",
			projectKey:    "TP",
			itemNumber:    1,
			want:          "http://localhost:3000/d/projects/TP/items/1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WorkItem(tt.baseURL, tt.namespaceSlug, tt.projectKey, tt.itemNumber); got != tt.want {
				t.Errorf("WorkItem() = %q, want %q", got, tt.want)
			}
		})
	}
}

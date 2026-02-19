package handler

import "testing"

func TestSanitizeContentType(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty defaults to octet-stream", "", "application/octet-stream"},
		{"image/png passes through", "image/png", "image/png"},
		{"application/pdf passes through", "application/pdf", "application/pdf"},
		{"text/plain passes through", "text/plain", "text/plain"},
		{"text/html blocked", "text/html", "application/octet-stream"},
		{"text/html with charset blocked", "text/html; charset=utf-8", "application/octet-stream"},
		{"text/javascript blocked", "text/javascript", "application/octet-stream"},
		{"application/javascript blocked", "application/javascript", "application/octet-stream"},
		{"application/xhtml+xml blocked", "application/xhtml+xml", "application/octet-stream"},
		{"image/svg+xml blocked", "image/svg+xml", "application/octet-stream"},
		{"unparseable defaults to octet-stream", ";;;", "application/octet-stream"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeContentType(tt.input); got != tt.want {
				t.Errorf("sanitizeContentType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSafeDownloadContentType(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"image/png is safe", "image/png", "image/png"},
		{"image/jpeg is safe", "image/jpeg", "image/jpeg"},
		{"audio/mpeg is safe", "audio/mpeg", "audio/mpeg"},
		{"video/mp4 is safe", "video/mp4", "video/mp4"},
		{"text/plain is safe", "text/plain", "text/plain"},
		{"application/pdf is safe", "application/pdf", "application/pdf"},
		{"application/json forced to octet-stream", "application/json", "application/octet-stream"},
		{"application/zip forced to octet-stream", "application/zip", "application/octet-stream"},
		{"text/csv forced to octet-stream", "text/csv", "application/octet-stream"},
		{"application/octet-stream stays", "application/octet-stream", "application/octet-stream"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := safeDownloadContentType(tt.input); got != tt.want {
				t.Errorf("safeDownloadContentType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSafeContentDisposition(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantSub  string // substring that must be present
	}{
		{"simple filename", "report.pdf", "report.pdf"},
		{"strips directory traversal", "../../../etc/passwd", "passwd"},
		{"strips path components", "uploads/secret/file.txt", "file.txt"},
		{"replaces quotes", `my"file.txt`, "my_file.txt"},
		{"replaces backslashes", `my\file.txt`, "my_file.txt"},
		{"replaces newlines", "my\nfile.txt", "my_file.txt"},
		{"replaces carriage returns", "my\rfile.txt", "my_file.txt"},
		{"dot becomes download", ".", "download"},
		{"dotdot becomes download", "..", "download"},
		{"always attachment", "file.txt", "attachment"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := safeContentDisposition(tt.filename)
			if got == "" {
				t.Fatal("safeContentDisposition returned empty string")
			}
			if !contains(got, tt.wantSub) {
				t.Errorf("safeContentDisposition(%q) = %q, want substring %q", tt.filename, got, tt.wantSub)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

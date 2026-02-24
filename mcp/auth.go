package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// Credentials stores the Taskwondo connection info.
type Credentials struct {
	URL    string `json:"url"`
	APIKey string `json:"api_key"`
}

// ConfigDir returns the configuration directory path.
func ConfigDir() string {
	if dir := os.Getenv("TASKWONDO_CONFIG_DIR"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".config", "taskwondo")
	}
	return filepath.Join(home, ".config", "taskwondo")
}

// LoadCredentials reads credentials from the config file.
func LoadCredentials() (*Credentials, error) {
	path := filepath.Join(ConfigDir(), "credentials.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read credentials: %w", err)
	}
	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("parse credentials: %w", err)
	}
	return &creds, nil
}

// SaveCredentials writes credentials to the config file with 0600 permissions.
func SaveCredentials(creds *Credentials) error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}
	path := filepath.Join(dir, "credentials.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write credentials: %w", err)
	}
	return nil
}

// DeleteCredentials removes the credentials file.
func DeleteCredentials() error {
	path := filepath.Join(ConfigDir(), "credentials.json")
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove credentials: %w", err)
	}
	return nil
}

// ResolveAuth determines the API key and base URL from env vars or credentials file.
func ResolveAuth() (baseURL, apiKey string, err error) {
	baseURL = os.Getenv("TASKWONDO_URL")
	apiKey = os.Getenv("TASKWONDO_API_KEY")

	if apiKey == "" {
		creds, loadErr := LoadCredentials()
		if loadErr != nil {
			return "", "", loadErr
		}
		if creds != nil {
			if apiKey == "" {
				apiKey = creds.APIKey
			}
			if baseURL == "" {
				baseURL = creds.URL
			}
		}
	}

	return baseURL, apiKey, nil
}

// BrowserLogin starts the browser-based login flow.
// It starts a temporary HTTP server, opens the browser to the Taskwondo authorize page,
// and waits for the callback with the API key.
func BrowserLogin(baseURL string) (*Credentials, error) {
	// Find a free port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	resultCh := make(chan *callbackResult, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		theme := q.Get("theme")
		brand := q.Get("brand")
		if brand == "" {
			brand = "Taskwondo"
		}

		if errMsg := q.Get("error"); errMsg != "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, renderCallbackPage(theme, brand, "error", errMsg))
			resultCh <- &callbackResult{err: fmt.Errorf("authorization denied: %s", errMsg)}
			return
		}
		key := q.Get("key")
		if key == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, renderCallbackPage(theme, brand, "error", "No API key received."))
			resultCh <- &callbackResult{err: fmt.Errorf("no API key received")}
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, renderCallbackPage(theme, brand, "success", ""))
		resultCh <- &callbackResult{apiKey: key}
	})

	srv := &http.Server{Handler: mux}
	go srv.Serve(listener)

	// Open browser
	authURL := fmt.Sprintf("%s/auth/cli/authorize?callback_port=%d&client_name=claude-code", baseURL, port)
	if err := openBrowser(authURL); err != nil {
		srv.Close()
		return nil, fmt.Errorf("open browser: %w (please visit manually: %s)", err, authURL)
	}

	// Wait for callback (timeout after 5 minutes)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var result *callbackResult
	select {
	case result = <-resultCh:
	case <-ctx.Done():
		srv.Close()
		return nil, fmt.Errorf("login timed out after 5 minutes")
	}

	srv.Close()

	if result.err != nil {
		return nil, result.err
	}

	creds := &Credentials{
		URL:    baseURL,
		APIKey: result.apiKey,
	}
	if err := SaveCredentials(creds); err != nil {
		return nil, fmt.Errorf("save credentials: %w", err)
	}
	return creds, nil
}

type callbackResult struct {
	apiKey string
	err    error
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		// Check for WSL
		if isWSL() {
			cmd = exec.Command("cmd.exe", "/c", "start", url)
		} else {
			cmd = exec.Command("xdg-open", url)
		}
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd.exe", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return cmd.Start()
}

func isWSL() bool {
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	return len(data) > 0 && (contains(string(data), "microsoft") || contains(string(data), "WSL"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// renderCallbackPage returns a self-contained HTML page styled to match the Taskwondo app.
// status is "success" or "error". For errors, msg contains the error description.
func renderCallbackPage(theme, brand, status, msg string) string {
	brand = html.EscapeString(brand)
	msg = html.EscapeString(msg)
	isDark := theme == "dark"

	// Colors matching the app's Tailwind theme
	var bgColor, cardBg, cardBorder, titleColor, subtitleColor, iconColor, iconBg string
	if isDark {
		bgColor = "#111827"    // gray-900
		cardBg = "#1f2937"     // gray-800
		cardBorder = "#374151" // gray-700
		titleColor = "#f3f4f6" // gray-100
		subtitleColor = "#9ca3af" // gray-400
	} else {
		bgColor = "#f9fafb"    // gray-50
		cardBg = "#ffffff"
		cardBorder = "#e5e7eb" // gray-200
		titleColor = "#111827" // gray-900
		subtitleColor = "#6b7280" // gray-500
	}

	var icon, heading, subtitle string
	if status == "success" {
		if isDark {
			iconColor = "#34d399" // green-400
			iconBg = "rgba(16, 185, 129, 0.1)"
		} else {
			iconColor = "#059669" // green-600
			iconBg = "rgba(16, 185, 129, 0.1)"
		}
		icon = `<svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="` + iconColor + `" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>`
		heading = "You're all set!"
		subtitle = "You are now logged in. You can close this tab."
	} else {
		if isDark {
			iconColor = "#f87171" // red-400
			iconBg = "rgba(239, 68, 68, 0.1)"
		} else {
			iconColor = "#dc2626" // red-600
			iconBg = "rgba(239, 68, 68, 0.1)"
		}
		icon = `<svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="` + iconColor + `" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/></svg>`
		heading = "Authorization failed"
		if msg != "" {
			subtitle = msg
		} else {
			subtitle = "Something went wrong. Please try again."
		}
	}

	return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>` + heading + ` — ` + brand + `</title>
<style>
  *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
  body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
    background-color: ` + bgColor + `;
    min-height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 1rem;
  }
  .card {
    background: ` + cardBg + `;
    border: 1px solid ` + cardBorder + `;
    border-radius: 0.5rem;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
    padding: 2.5rem 2rem;
    max-width: 28rem;
    width: 100%;
    text-align: center;
  }
  .icon-wrap {
    width: 5rem;
    height: 5rem;
    margin: 0 auto 1.5rem;
    border-radius: 50%;
    background: ` + iconBg + `;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  h1 {
    color: ` + titleColor + `;
    font-size: 1.25rem;
    font-weight: 600;
    margin-bottom: 0.5rem;
  }
  .subtitle {
    color: ` + subtitleColor + `;
    font-size: 0.875rem;
    line-height: 1.5;
  }
  .brand {
    color: ` + subtitleColor + `;
    font-size: 0.75rem;
    margin-top: 1.5rem;
  }
</style>
</head>
<body>
  <div class="card">
    <div class="icon-wrap">` + icon + `</div>
    <h1>` + heading + `</h1>
    <p class="subtitle">` + subtitle + `</p>
    <p class="brand">` + brand + `</p>
  </div>
</body>
</html>`
}

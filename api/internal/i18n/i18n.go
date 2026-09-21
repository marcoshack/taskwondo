package i18n

import (
	"embed"
	"encoding/json"
	"strings"
)

//go:embed translations/*.json
var translationFS embed.FS

// translations maps language code to key-value pairs.
var translations map[string]map[string]string

func init() {
	translations = make(map[string]map[string]string)
	entries, _ := translationFS.ReadDir("translations")
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		lang := strings.TrimSuffix(entry.Name(), ".json")
		data, err := translationFS.ReadFile("translations/" + entry.Name())
		if err != nil {
			continue
		}
		var m map[string]string
		if err := json.Unmarshal(data, &m); err != nil {
			continue
		}
		translations[lang] = m
	}
}

// T returns the translation for the given key in the given language.
// Falls back to English if the key is not found in the requested language.
// Placeholders like {{name}} are replaced using variadic key-value pairs.
func T(lang, key string, args ...string) string {
	val := get(lang, key)
	for i := 0; i+1 < len(args); i += 2 {
		val = strings.ReplaceAll(val, "{{"+args[i]+"}}", args[i+1])
	}
	return val
}

// Negotiate resolves a language from an HTTP Accept-Language header value.
// Candidates are checked in header order; the first supported primary subtag
// wins. Falls back to "en" when nothing matches.
func Negotiate(acceptLanguage string) string {
	for _, part := range strings.Split(acceptLanguage, ",") {
		if i := strings.IndexByte(part, ';'); i >= 0 {
			part = part[:i]
		}
		primary := strings.TrimSpace(part)
		if i := strings.IndexByte(primary, '-'); i >= 0 {
			primary = primary[:i]
		}
		primary = strings.ToLower(primary)
		if _, ok := translations[primary]; ok {
			return primary
		}
	}
	return "en"
}

func get(lang, key string) string {
	if m, ok := translations[lang]; ok {
		if v, ok := m[key]; ok {
			return v
		}
	}
	if lang != "en" {
		if m, ok := translations["en"]; ok {
			if v, ok := m[key]; ok {
				return v
			}
		}
	}
	return key
}

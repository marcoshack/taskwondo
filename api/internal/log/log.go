package log

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Setup configures the global zerolog logger with the given level, output
// format ("text" for console, anything else for JSON), and service name.
func Setup(level, format, service string) {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(lvl)
	if format == "text" {
		writer := zerolog.ConsoleWriter{
			Out: os.Stderr,
			// Render the service field as "[name]" right after the level so
			// interleaved api/worker lines in `make dev` are easy to tell apart.
			PartsOrder: []string{
				zerolog.TimestampFieldName,
				zerolog.LevelFieldName,
				zerolog.CallerFieldName,
				"service",
				zerolog.MessageFieldName,
			},
			FormatPartValueByName: func(i interface{}, name string) string {
				if name != "service" {
					return ""
				}
				s, ok := i.(string)
				if !ok || s == "" {
					return ""
				}
				return colorizeService(s)
			},
		}
		log.Logger = zerolog.New(writer).With().Timestamp().Caller().Str("service", service).Logger()
	} else {
		log.Logger = zerolog.New(os.Stderr).With().Timestamp().Str("service", service).Logger()
	}
}

// serviceColors maps a service name to a bold ANSI color code so the "[name]"
// tag stands out and is consistent per service in the console writer. Unknown
// services fall back to a stable color derived from the name.
var serviceColors = map[string]int{
	"api":    36, // cyan
	"worker": 33, // yellow
}

// colorizeService wraps "[name]" in a bold ANSI color chosen per service.
func colorizeService(s string) string {
	color, ok := serviceColors[s]
	if !ok {
		// Stable fallback across the bright-color range (91–96).
		var sum int
		for _, r := range s {
			sum += int(r)
		}
		color = 91 + sum%6
	}
	return fmt.Sprintf("\x1b[1;%dm[%s]\x1b[0m", color, s)
}

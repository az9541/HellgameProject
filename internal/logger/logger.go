package logger

import (
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
)

func Setup(isDev bool) {
	var handler slog.Handler

	options := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}

	if isDev {
		handler = tint.NewHandler(os.Stdout, &tint.Options{
			Level:      slog.LevelDebug,
			TimeFormat: time.TimeOnly,
			// Перехватываем атрибуты перед выводом
			ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
				// Если ключ называется "event_type"
				if a.Key == "event_type" && a.Value.Kind() == slog.KindString {
					val := a.Value.String()

					// Базовые ANSI-цвета
					reset := "\033[0m"
					red := "\033[31m"
					green := "\033[32m"
					yellow := "\033[33m"
					magenta := "\033[35m"

					// Раскрашиваем в зависимости от значения типа ивента
					switch val {
					case "WAR_STARTED", "WAR_UPDATE":
						a.Value = slog.StringValue(red + val + reset)
					case "WAR_AVOIDED", "WWAR_ENDED":
						a.Value = slog.StringValue(green + val + reset)
					case "TAKEOVER_ATTEMPT":
						a.Value = slog.StringValue(yellow + val + reset)
					case "TAKEOVER_FAILED":
						a.Value = slog.StringValue(magenta + val + reset)
					}
				}
				return a
			},
		})
	} else {
		options.Level = slog.LevelInfo
		handler = slog.NewJSONHandler(os.Stdout, options)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
}

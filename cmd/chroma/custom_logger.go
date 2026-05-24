package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type CustomHandler struct {
	level  slog.Level
	attrs  []slog.Attr
	groups []string
}

func (h *CustomHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *CustomHandler) Handle(_ context.Context, r slog.Record) error {
	// 1. Date Time Stamp
	timeStr := r.Time.Format("2006-01-02 15:04:05")

	// 2. Exatract Source (File name: struct#Method)
	location := "unknown"

	if r.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		frame, _ := fs.Next()
		file := filepath.Base(frame.File)
		fullFunctionName := filepath.Base(frame.Function) // e.g. "main.(*Client).GetData"

		parts := strings.Split(fullFunctionName, ".")
		if len(parts) >= 2 && strings.Contains(fullFunctionName, "(") {
			// It's a method: e.g. "Client#GetData"
			structPart := strings.Trim(parts[len(parts)-2], "()*")
			methodPart := parts[len(parts)-1]
			location = fmt.Sprintf("%s: %s#%s", file, structPart, methodPart)
		} else {
			// It's a plain function : just show file and function name
			funcName := parts[len(parts)-1]
			location = fmt.Sprintf("%s#%s", file, funcName)
		}
	}
	// 3. Final Output - Include log level
	// Format: [LEVEL] timestamp:file:location:message
	levelStr := r.Level.String()
	_, _ = fmt.Fprintf(os.Stdout, "[%s] %s:%s:%s\n", levelStr, timeStr, location, r.Message)

	// Merge handler attrs + record attrs
	allAttrs := make([]slog.Attr, 0, len(h.attrs))
	allAttrs = append(allAttrs, h.attrs...)

	// Collect record attrs
	r.Attrs(func(a slog.Attr) bool {
		allAttrs = append(allAttrs, a)
		return true
	})

	// Apply group prefix
	groupPrefix := ""
	if len(h.groups) > 0 {
		groupPrefix = strings.Join(h.groups, ".") + "."
	}

	// Print attributes with optional prefix
	for _, a := range allAttrs {
		key := a.Key
		if groupPrefix != "" {
			key = groupPrefix + a.Key
		}

		if a.Value.String() != "" {
			fmt.Printf("	└ %s: %v\n", key, a.Value)
		}
	}

	return nil
}

func (h *CustomHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &CustomHandler{
		level:  h.level,
		attrs:  append(h.attrs, attrs...),
		groups: h.groups,
	}
}

func (h *CustomHandler) WithGroup(name string) slog.Handler {
	return &CustomHandler{
		level:  h.level,
		attrs:  h.attrs,
		groups: append(h.groups, name),
	}
}

func InitLogger(level slog.Level, format string) {
	var handler slog.Handler

	if format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:     level,
			AddSource: true,
		})
	} else {
		handler = &CustomHandler{level: level}
	}

	slog.SetDefault(slog.New(handler))
}

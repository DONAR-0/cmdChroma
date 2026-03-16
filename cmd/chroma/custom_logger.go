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
	level slog.Level
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
	// 3. Final Output
	// Format: Date Time Stamp : File Name : Location : actual log Value
	_, _ = fmt.Fprintf(os.Stdout, "%s:%s:%s\n", timeStr, location, r.Message)
	// Print extra attributes (like errors) on a new Line
	r.Attrs(func(a slog.Attr) bool {
		if a.Value.String() != "" {
			fmt.Printf("	└ %s: %v\n", a.Key, a.Value)
		}
		return true
	})
	return nil
}

func (h *CustomHandler) WithAttrs(attrs []slog.Attr) slog.Handler { return h }
func (h *CustomHandler) WithGroup(name string) slog.Handler       { return h }

func InitLogger() {
	handler := &CustomHandler{level: slog.LevelInfo}
	slog.SetDefault(slog.New(handler))
}

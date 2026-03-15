// Package version provides build-time version information.
//
// This package is designed to be updated via `go generate`.
// Run `go generate ./...` to regenerate the build metadata.

package version

//go:generate go run gen.go

// Version is the current semantic version of the binary.
// It defaults to "0.0.1" and can be overridden via ldflags at build time.
var Version = "0.0.1"

# Contributing to cmdChroma

Thank you for your interest in contributing! This document provides guidelines and information for contributors.

---

## Getting Started

### Prerequisites

- **Go**: 1.21 or higher ([download](https://go.dev/dl/))
- **Git**: For version control
- **Docker**: Required for running ChromaDB and integration tests (optional but recommended)
- **GNU Make**: For using build targets (optional but convenient)

### Development Environment Setup

1. **Clone the repository**
   ```bash
   git clone https://github.com/DONAR-0/cmdChroma.git
   cd cmdChroma
   ```

2. **Download dependencies and models** (one-time setup)
   ```bash
   # This script downloads the ONNX model, tokenizer, and ONNX Runtime library
   .ci/scripts/setup.sh
   ```

3. **Build the CLI**
   ```bash
   make build
   # Binary will be at ./cmdChroma
   ```

4. **Run ChromaDB** (required for integration tests)
   ```bash
   docker run -d -p 8000:8000 --name chromadb chromadb/chroma
   ```

5. **Verify installation**
   ```bash
   ./cmdChroma ping
   ```

---

## Development Workflow

### Making Changes

1. **Create a branch** for your changes
   ```bash
   git checkout -b my-feature-branch
   ```

2. **Make your changes** following the code style guidelines (see below)

3. **Run tests** to ensure nothing breaks
   ```bash
   # Format code
   make fmt

   # Lint (staticcheck, go vet)
   make lint

   # Build
   make build

   # Unit tests
   make test

   # Integration tests (requires ChromaDB running)
   make venom
   ```

   Or run everything in sequence:
   ```bash
   make dev   # fmt → lint → build → test → venom (integration tests)
   ```

4. **Commit your changes** with a clear commit message
   ```bash
   git add .
   git commit -m "feat: add support for Parquet import streaming"
   ```
   We follow [Conventional Commits](https://www.conventionalcommits.org/) format (feat, fix, docs, test, refactor, etc.)

5. **Push to your fork** and open a Pull Request
   ```bash
   git push origin my-feature-branch
   ```

### Code Style

- **Formatting**: Always run `make fmt` before committing. This uses `go fmt` and `gofmt`.
- **Linting**: Run `make lint` to catch common issues. The linter is `golangci-lint` (or `go vet` as fallback).
- **Imports**: Gofmt handles import ordering and grouping.
- **Error handling**: Return wrapped errors (`fmt.Errorf("...: %w", err)`) to preserve context.
- **Logging**: Use `log/slog` with structured logging (key-value pairs). Use appropriate levels: Debug, Info, Warn, Error.
- **Comments**: All exported identifiers must have doc comments. See [Go Documentation Best Practices](docs/plans/documentation/research.md).

---

## Testing

### Unit Tests

Run with:
```bash
go test ./...
# or
make test
```

Add tests for new functionality in the appropriate package under `*_test.go`.

### Integration Tests

cmdChroma uses [Venom](https://github.com/ovh/venom) for end-to-end tests written in YAML.

**Run all integration tests:**
```bash
make venom
```

**Run a subset:**
```bash
venom -t .ci/tests/commands/
venom -t .ci/tests/smoke/
```

**During development:** You can run individual test files. See [TESTING.md](TESTING.md) for details.

---

## Documentation

### Command Documentation

CLI command help text is defined in `cmd/chroma/descriptions.go`. Each command's description should:
- Begin with a clear summary of what the command does
- List key steps or behaviors in a bullet list
- Provide 3-5 practical examples with explanations
- Mention required dependencies (e.g., "Ollama must be running")

### README Updates

When adding new features or changing behavior, update `README.md`:
- Add or update command examples in the **Command Reference** section
- Update the **Features** list if applicable
- Add troubleshooting tips if new failure modes are possible

### Code Documentation

All exported functions, types, and methods must have doc comments. Follow these guidelines:
- **Package comments**: Start with "Package `<name>`" explaining the package's purpose.
- **Function comments**: Start with the function name implicitly: `// NewClient creates...` not `// This function creates...`
- **Parameter descriptions**: Only needed if not obvious from the name. Explain semantics, units, defaults, constraints.
- **Return values**: Describe what each return represents, especially errors.
- **Examples**: Add `Example*` functions to demonstrate usage. They serve as both documentation and tests.

See `docs/plans/documentation/research.md` for detailed Go documentation best practices.

---

## Architecture Overview

cmdChroma follows a layered architecture:

```
cmd/chroma/          → CLI layer (commands, flags, output formatting)
internal/service/    → Business logic layer (orchestration, validation)
internal/client/     → HTTP client for ChromaDB REST API
internal/onnx/       → ONNX-based embedding generation
internal/llm/        → LLM provider abstractions (Ollama, NVIDIA NIM)
internal/ingest/     → File parsing and streaming (JSONL, Parquet)
internal/config/     → Configuration cascade (flags, env, file)
internal/errors/     → Canonical error definitions
```

When adding new functionality, place code in the appropriate layer and respect boundaries:
- CLI handlers should delegate to `service`; they handle validation only.
- Service depends on `client` and `onnx` interfaces, not concrete implementations.
- Use interface types for dependency injection (testability).

---

## Pull Request Process

1. **Ensure tests pass**: CI will run unit and integration tests automatically. Fix any failures.
2. **Update documentation**: Include doc comments, README updates, and examples as needed.
3. **Keep PRs focused**: One feature or fix per PR. Avoid unrelated changes.
4. **Write a clear PR description**:
   - What problem does this solve?
   - How does it work at a high level?
   - Any breaking changes? Migration steps?
   - Screenshots or output examples if applicable
5. **Address review feedback**: Respond to code review comments and make requested changes.
6. **Squash commits** if requested (we prefer clean commit history).

---

## Reporting Issues

Found a bug or have a feature request? Open an issue on GitHub:

- **Bug reports**: Include steps to reproduce, expected vs actual behavior, environment details (OS, Go version, cmdChroma version), and logs.
- **Feature requests**: Explain the use case, proposed solution, and alternatives considered.
- **Questions**: Use GitHub Discussions (if enabled) or the issue tracker.

---

## Code of Conduct

This project follows the [Contributor Covenant](https://www.contributor-covenant.org/). Please be respectful and constructive in all interactions.

---

## Additional Resources

- [TESTING.md](TESTING.md) - Comprehensive testing guide
- [README.md](README.md) - User-facing documentation
- [docs/plans/documentation/research.md](docs/plans/documentation/research.md) - Go documentation best practices
- Effective Go: https://golang.org/doc/effective_go
- Go Code Review Comments: https://github.com/golang/go/wiki/CodeReviewComments

---

Thank you for contributing to cmdChroma! 🚀

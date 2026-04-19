# 🚀 cmdChroma CLI

A high-performance Go-based Command Line Interface for managing **ChromaDB** collections and performing local vector embeddings. This tool is designed for developers building **RAG (Retrieval-Augmented Generation)** pipelines who want to keep their data and AI processing entirely on their local machine.

[![Go CI](https://github.com/donar0/cmdChroma/actions/workflows/ci.yml/badge.svg)](https://github.com/donar0/cmdChroma/actions/workflows/ci.yml)
[![Integration Tests](https://github.com/donar0/cmdChroma/actions/workflows/integration.yml/badge.svg)](https://github.com/donar0/cmdChroma/actions/workflows/integration.yml)
[![Docker Build](https://github.com/donar0/cmdChroma/actions/workflows/docker.yml/badge.svg)](https://github.com/donar0/cmdChroma/actions/workflows/docker.yml)
[![Release](https://github.com/donar0/cmdChroma/actions/workflows/release.yml/badge.svg)](https://github.com/donar0/cmdChroma/actions/workflows/release.yml)
[![codecov](https://codecov.io/gh/donar0/cmdChroma/branch/main/graph/badge.svg)](https://codecov.io/gh/donar0/cmdChroma)

---

## ✨ Features

* **Local AI Embedding:** Integrated ONNX Runtime generates embeddings locally using `all-MiniLM-L6-v2`. **No OpenAI API keys or internet required.**
* **Batch Operations:** High-speed batch ingestion (`add`) and multi-query searching (`query`) to maximize CPU efficiency.
* **WSL & Linux Optimized:** Tailored for Windows Subsystem for Linux (WSL) with automated path resolution and cross-platform build support.
* **Dataset Ingestion:** Built-in logic to stream and import large datasets (like Wikipedia) from JSONL formats.
* **Scalable Architecture:** Built with `urfave/cli/v3` for a robust, flag-driven user experience.

---

## 🛠️ Installation & Setup

## 🚀 Getting Started
1. Clone the repo: `git clone ...`
2. Run the setup script to download AI models: `./.ci/scripts/setup.sh`
3. Build the CLI: `make build-wsl`

### 1. Prerequisites
* **Go:** 1.21 or higher.
* **ChromaDB:** Running locally (typically via Docker).
  ```bash
  docker run -p 8000:8000 chromadb/chroma
  ```
* **ONNX Runtime:** Ensure the shared library (`libonnxruntime.so`) is available in your models directory.

### 2. Project Structure
Ensure your models are placed in the `models/` directory relative to the binary:
```
.
├── cmd/
│   └── chroma/
│       ├── main.go
│       ├── handlers.go
│       └── definitions.go
├── internal/
│   ├── client/
│   └── onnx/
├── models/
│   ├── all-MiniLM-L6-v2/
│   │   ├── model.onnx
│   │   └── tokenizer.json
│   └── onnx_runtime/
│       └── lib/libonnxruntime.so
└── Makefile
```

### 3. Docker Installation
You can also run cmdChroma via Docker:

```bash
# Pull the latest image
docker pull donar0/cmdchroma:latest

# Run the CLI
docker run --rm donar0/cmdchroma:latest --help
```

For local development with ChromaDB:
```bash
# Start ChromaDB
docker run -d -p 8000:8000 --name chromadb chromadb/chroma

# Run cmdChroma with network access to ChromaDB
docker run --rm --network host donar0/cmdchroma:latest test
```

### 4. Build
Use the provided Makefile to ensure proper formatting and WSL compatibility:
```bash
make build-wsl
```

### 4. Docker Executable (Optional)
If you prefer running `cmdChroma` inside a container, this repo includes a Docker build helper.

Build the image:
```bash
./.ci/docker/build.sh
```

Run `cmdChroma` inside the container (uses `docker exec`):
```bash
./.ci/docker/exec.sh --help
```

If you want to export the built binary rather than a container image, enable the export mode:
```bash
DOCKER_OUTPUT=1 ./.ci/docker/build.sh
```

> **Note:** Full embedding support requires the native `libtokenizers` library.
> If you do not have `libtokenizers` installed, the CLI will still build and run
> (using a no-op placeholder embedder), but embedding-based commands will use
> zero vectors.

---

## 📖 Usage Examples

### Create a Collection
```bash
./cmdChroma create wikipedia_simple
```

### Add Documents (Batch)
Add multiple documents in a single command. The CLI vectorizes them locally before sending them to Chroma.
```bash
./cmdChroma add wikipedia_simple \
  --doc "Go is a statically typed, compiled high-level language." \
  --doc "ChromaDB stores embeddings for semantic search."
```

### Batch Query
Search for multiple topics simultaneously.
```bash
./cmdChroma query wikipedia_simple \
  -q "What is Go?" \
  -q "How do vector databases work?" \
  --n-results 3
```

### Version Info
The CLI includes build metadata in `--version` output:
```bash
./cmdChroma --version
# -> cmdChroma version (git <commit>, built <timestamp>)
```

### Dataset Import
Import a JSONL file exported from Hugging Face or other sources:
```bash
./cmdChroma import wikipedia_simple ./data/wiki_subset.jsonl
```

---

## 🧪 Testing

### Unit Tests
Run the Go unit tests for the entire module:
```bash
go test ./...
# or via Makefile
make test
```

### Code Coverage
Test coverage is automatically calculated and uploaded to Codecov on each push. View the latest coverage report [here](https://codecov.io/gh/donar0/cmdChroma).

### Linting
The codebase is linted using `golangci-lint` in CI to ensure code quality and consistency.

### Integration Tests (Venom)
Run the full integration suite using Venom:
```bash
make venom
# or directly
./.ci/scripts/run-venom.sh
```

### Code Generation (`go generate`)
This repository includes a small code generator that produces build metadata in `internal/version/version_gen.go`.

To regenerate build metadata (timestamp + git commit), run:
```bash
go generate ./...
# or via Makefile
make generate
```

You can then access generated constants from code via:
```go
import "github.com/donar0/cmdChroma/internal/version"

fmt.Println(version.BuildDate, version.GitCommit)
```

---

## 📜 License & Attribution

This project is licensed under the Apache License 2.0.

### Third-Party Attributions
* **ChromaDB:** The AI-native open-source embedding database.
* **ONNX Runtime:** High-performance ML inferencing engine by Microsoft.
* **Hugging Face:** For the `all-MiniLM-L6-v2` transformer model.

Copyright © 2026 DONAR-0. Licensed under the Apache License, Version 2.0.
# 🚀 cmdChroma CLI

A high-performance, user-friendly command-line tool for **ChromaDB** with local AI embeddings. Designed for developers building **RAG (Retrieval-Augmented Generation)** pipelines who want to keep their data and AI processing entirely local.

[![Go CI](https://github.com/DONAR-0/cmdChroma/actions/workflows/ci.yml/badge.svg)](https://github.com/DONAR-0/cmdChroma/actions/workflows/ci.yml)
[![Integration Tests](https://github.com/DONAR-0/cmdChroma/actions/workflows/integration.yml/badge.svg)](https://github.com/DONAR-0/cmdChroma/actions/workflows/integration.yml)
[![Release](https://github.com/DONAR-0/cmdChroma/actions/workflows/release.yml/badge.svg)](https://github.com/DONAR-0/cmdChroma/actions/workflows/publish-ghcr.yml)
[![codecov](https://codecov.io/gh/DONAR-0/cmdChroma/branch/main/graph/badge.svg)](https://codecov.io/gh/DONAR-0/cmdChroma)

---

## 🌟 Quick Start (30 seconds)

1. **Start ChromaDB**
   ```bash
   docker run -d -p 8000:8000 --name chromadb chromadb/chroma
   ```

2. **Test connection**
   ```bash
   ./cmdChroma ping
   ```

3. **Create a collection and add data**
   ```bash
   ./cmdChroma create my_docs
   ./cmdChroma add my_docs --doc "Your document text here"
   ```

4. **Search your data**
   ```bash
   ./cmdChroma query my_docs --query "search terms"
   ```

That's it! 🎉

---

## ✨ Features

- **Local AI Embedding**: All embeddings generated locally using ONNX Runtime (`all-MiniLM-L6-v2`). No OpenAI API keys or internet required.
- **Batch Operations**: High-speed batch ingestion and multi-query searching for maximum efficiency.
- **Intuitive CLI**: Designed following [clig.dev](https://clig.dev) guidelines for a consistent, user-friendly experience.
- **Rich Error Messages**: Actionable hints when things go wrong.
- **Dataset Import**: Stream large datasets from JSONL files with progress reporting.
- **RAG-Ready**: Built-in chat command for Retrieval-Augmented Generation (supports both Ollama and NVIDIA NIM).
- **Cross-Platform**: Optimized for WSL/Linux with automated path resolution.

---

## 🏗️ How It Works

cmdChroma is built with a layered architecture for clean separation of concerns:

```
┌─────────────────────────────────────────────────────────────┐
│                         CLI Layer                           │
│  (Commands • Flags • Validation • Output Formatting)      │
├─────────────────────────────────────────────────────────────┤
│                    Service Layer                           │
│  (Business Logic • Orchestration • Error Handling)        │
├─────────────────────────────────────────────────────────────┤
│                    Client Layer                            │
│  (HTTP Client • API Calls • Response Unmarshaling)        │
├─────────────────────────────────────────────────────────────┤
│                External Dependencies                       │
│   ChromaDB Server  │  ONNX Runtime  │  LLM Provider       │
└─────────────────────────────────────────────────────────────┘
```

### Key Components

- **CLI Layer** (`cmd/chroma/`): User-facing commands built with [urfave/cli](https://github.com/urfave/cli). Handles input validation, flag parsing, and output formatting (text/JSON/TUI).
- **Service Layer** (`internal/service/`): Contains business logic. Orchestrates client operations and embedding generation. Manages batching and streaming.
- **Client Layer** (`internal/client/`): HTTP client that communicates with ChromaDB's REST API. Handles authentication, retries, and request/response serialization.
- **Embedder** (`internal/onnx/`): Local ONNX Runtime integration for generating vector embeddings. Uses the `all-MiniLM-L6-v2` model, producing 384-dimensional vectors.
- **LLM Providers** (`internal/llm/`): Pluggable backends for chat functionality (Ollama for local, NVIDIA NIM for cloud).

### Data Flow Example: Adding Documents

```
User runs: chroma add my_collection --doc "Hello world"
   ↓
CLI parses flags, validates collection name
   ↓
Service calls embedder.Embed("Hello world") → []float32{0.1, 0.2, ...}
   ↓
Client sends HTTP POST to /api/v2/.../add with embeddings
   ↓
ChromaDB stores documents + vectors
   ↓
CLI prints success message
```

### Why Local Embeddings?

By generating embeddings locally with ONNX Runtime, cmdChroma:
- **Protects privacy**: No data leaves your machine
- **Reduces costs**: No per-embedding API fees
- **Works offline**: No internet required after initial model download
- **Scales freely**: No rate limits or usage quotas

---

## 📦 Installation

### Prerequisites
- **Go**: 1.21 or higher (for building from source)
- **ChromaDB**: Running locally (see quick start above)
- **ONNX Runtime**: `libonnxruntime.so` and model files (automatically handled by setup script)

### Build from Source
```bash
# Clone the repository
git clone https://github.com/DONAR-0/cmdChroma.git
cd cmdChroma

# Download AI models (one-time setup)
./.ci/scripts/setup.sh

# Build the CLI
make build

# Or for WSL/Linux specifically
make build-linux
```

The binary will be at `./cmdChroma`.

### Docker (Easiest)
```bash
# Pull pre-built image from GitHub Container Registry
docker pull ghcr.io/donar-0/cmdchroma:latest

# Run (with network access to ChromaDB)
docker run --rm --network host ghcr.io/donar-0/cmdchroma:latest ping
```

### Project Structure
Your models should be placed in `models/` relative to the binary:
```
.
├── cmdChroma              # built binary
└── models/
    ├── all-MiniLM-L6-v2/
    │   ├── model.onnx
    │   └── tokenizer.json
    └── onnx_runtime/
        └── lib/libonnxruntime.so
```

> **Note**: The setup script automatically downloads these files to the correct location.

---

## 🎯 Command Reference

### Core Commands

| Command | Purpose | Common Use |
|---------|---------|------------|
| `ping` | Test ChromaDB connectivity | Verify server is running |
| `create` | Create a collection | Before adding documents |
| `add` | Add documents with embedding | Ingest single or batch documents |
| `query` | Semantic search | Find similar documents |
| `import` | Bulk import from JSONL | Load datasets |
| `chat` | RAG Q&A | Ask questions about your data |
| `records` | List documents | Inspect collection contents |
| `databases` | List databases | Explore tenant |
| `collections` | List collections | See available collections |
| `tenant` | Show current tenant | Verify tenant config |
| `delete` | Delete collection | Remove collection |
| `delete-records` | Delete documents by ID | Remove specific records |
| `config` | Configuration management | Initialize config, show settings |

#### `ping` — Test Connection
Verify connectivity to your ChromaDB instance.

**Example output:**
```
Successfully connected to ChromaDB
   Server: localhost:8000
   Tenant: default_tenant
```

```bash
# Test default connection
chroma ping

# Test with custom host/port
chroma ping --host 192.168.1.10 --port 8080

# Set a shorter timeout (default: 10s)
chroma ping --timeout 5

# Verbose mode for debugging
chroma ping --verbose
```

#### `create` — Create Collection
Create a new collection to store your documents.

**Example output:**
```
✅ Collection 'my_collection' created (ID: 123e4567-e89b-12d3-a456-426614174000)
```

```bash
# Simple collection with default tenant/database
chroma create my_collection

# Create in specific tenant and database
chroma create my_collection --tenant my_tenant --database my_db

# Auto-create database if it doesn't exist
chroma create my_collection --create-db

# Use a custom collection metadata (TODO if supported)
```

#### `add` — Add Documents
Add one or more documents with automatic local embedding.

**Example output:**
```
✅ Successfully added 3 documents to 'my_collection'
```

```bash
# Single document
chroma add my_collection --doc "The capital of France is Paris."

# Multiple documents (batch)
chroma add my_collection \
  --doc "Go is a compiled language." \
  --doc "Python is interpreted." \
  --doc "JavaScript runs in browsers."

# With custom IDs (must match document count)
chroma add my_collection \
  --doc "Document 1" --id "doc-1" \
  --doc "Document 2" --id "doc-2"

# Upsert (add or update if exists)
chroma add my_collection --doc "Updated content" --id "existing-id" --upsert

# From a file using xargs (useful for many docs)
cat docs.txt | xargs -I {} chroma add my_collection --doc "{}"

# Batch size control (advanced)
chroma add my_collection --doc "text" --batch-size 50  # default: 100
```

#### `query` — Search Documents
Perform semantic search with natural language queries.

**Example output:**
```
🔍 Search results for collection 'my_collection':

Query 1: What is Go?
------------------------------------------------------------
  [1] Distance: 0.2345
      ID: doc-123
      Content: Go is a compiled language...

  [2] Distance: 0.4567
      ID: doc-456
      Content: Programming in Go involves...

Query 2: How to use vectors?
------------------------------------------------------------
...
```

```bash
# Single query
chroma query my_collection --query "What is Go programming?"

# Multiple queries (much faster than running separately)
chroma query my_collection \
  --query "How to use vectors?" \
  --query "What is RAG?" \
  --query "Explain embeddings"

# Get more results
chroma query my_collection --query "AI concepts" --n-results 10

# Use verbose logging for timing/debug
chroma query my_collection --query "test" --verbose
```

#### `import` — Import JSONL File
Bulk import from JSONL files (e.g., Hugging Face datasets).

**Example output:**
```
📥 Importing from 'data.jsonl' to collection 'my_collection'
   Batch size: 100
   Limit: 1000 documents

✅ Import completed in 45s
```

```bash
# Basic import (expects default fields: 'text' or 'content', and 'id')
chroma import my_collection data.jsonl

# Custom field mapping (your JSON uses different field names)
chroma import my_collection data.jsonl \
  --field-content "content" \
  --field-id "uuid"

# Import with metadata extraction and limit
chroma import my_collection data.jsonl \
  --field-metadata "author" \
  --field-metadata "category" \
  --all-metadata \
  --n-ingest 1000 \
  --batch-size 500

# Import Parquet file (TODO - verify format support)
chroma import my_collection data.parquet --batch-size 200

# Show progress and timing with verbose
chroma import my_collection large_data.jsonl --verbose
```


#### `chat` — RAG Q&A
Ask questions about your collection using AI-powered retrieval (supports Ollama and NVIDIA NIM).

**Example output:**
```
🤖 Querying collection 'my_collection' with: "What is Go?"

💭 Generating answer...
🤖 AI Response:
--------------------
Go is a statically typed, compiled programming language designed at Google.
It emphasizes simplicity, concurrency, and performance.
```

**Ollama (local):**
```bash
# Requires Ollama running (default: http://localhost:11434)
chroma chat my_collection "What are the main topics?"

# Use a specific model (must be pulled first: ollama pull llama2)
chroma chat my_collection "Summarize this" --llm-model llama2

# Retrieve more context for complex questions
chroma chat my_collection "Explain in detail" --n-results 10

# Set distance threshold to ignore irrelevant matches
chroma chat my_collection "Explain this" --distance-threshold 20
```

**NVIDIA NIM (cloud API):**
```bash
# Set your NVIDIA API key (required)
export NVIDIA_API_KEY="your-api-key-here"

# Use a NIM model (prefix with nim://)
# Check available models: curl -H "Authorization: Bearer $NVIDIA_API_KEY" https://integrate.api.nvidia.com/v1/models
chroma chat my_collection "What are the main topics?" --llm-model nim://mistralai/mistral-7b-instruct-v0.3

# Custom NIM endpoint (default: https://integrate.api.nvidia.com/v1)
chroma chat my_collection "Explain this" --llm-model nim://meta/llama-3.1-8b-instruct --nim-url https://custom.endpoint.com/v1

# Combine with other flags
chroma chat my_collection "Summarize" --llm-model nim://... --n-results 5 --distance-threshold 15
```

#### `records` — List Documents
Inspect documents in a collection.
```bash
chroma records my_collection
chroma records my_collection | head -n 20  # Paginate
```

#### `databases` — List Databases
View all databases in the current tenant.
```bash
chroma databases
chroma databases --tenant custom_tenant
```

#### `collections` — List Collections
View all collections in the current database.
```bash
chroma collections
chroma collections --database my_db
```

#### `tenant` — Show Current Tenant
Check tenant configuration.
```bash
chroma tenant
chroma tenant --tenant my_tenant
```

#### `delete` — Delete Collection
Permanently remove a collection and all its data.
```bash
chroma delete my_collection
chroma delete my_collection --tenant my_tenant --database my_db
```
⚠️ **This operation cannot be undone.**

#### `delete-records` — Delete Documents
Remove specific documents from a collection by their IDs.
```bash
# Delete a single document
chroma delete-records my_collection --id doc-123

# Delete multiple documents
chroma delete-records my_collection --id doc-1 --id doc-2 --id doc-3
```

#### `config` — Configuration Management
Manage configuration files and view effective settings.
```bash
# Show current effective configuration (merged from all sources)
chroma config show

# Create a default config file in current directory
chroma config init

# Create a global config file in ~/.config/cmdChroma/
chroma config init --global

# Show configuration as JSON
chroma config show --output json
```

#### `doctor` — Diagnose Issues
Run diagnostics to verify your installation and configuration.
```bash
# Run all diagnostic checks
chroma doctor

# Save diagnostic report to a file
chroma doctor --output report.txt
```

---

## 🔧 Global Flags

These flags are available on all commands:

| Flag | Alias | Environment | Default | Description |
|------|-------|-------------|---------|-------------|
| `--host` | `-H` | `CHROMA_HOST` | `localhost` | ChromaDB server host |
| `--port` | `-p` | `CHROMA_PORT` | `8000` | ChromaDB server port |
| `--tenant` | | `TENANT`, `CHROMA_TENANT` | `default_tenant` | Tenant name |
| `--database` | | `DATABASE`, `CHROMA_DATABASE` | `default_database` | Database name |
| `--collection` | | `COLLECTION`, `CHROMA_COLLECTION` | (none) | Default collection (can override per command) |
| `--verbose` | `-v` | | `false` | Enable debug logging |

---

## ⚙️ Configuration

cmdChroma provides three ways to configure its behavior, applied in this precedence order (higher overrides lower):

1. **CLI Flags** (highest priority) - Set per command invocation
2. **Environment Variables** - Process-level, ideal for containers
3. **Configuration File** - Persistent defaults in YAML

### Configuration File

Create a `.cmdChroma.yaml` in your project directory or in `~/.config/cmdChroma/config.yaml` for user-wide defaults.

**Example `.cmdChroma.yaml`:**
```yaml
chroma:
  host: localhost
  port: 8000
  tenant: default_tenant
  database: default_database
  collection: my_default_collection  # optional default

logging:
  level: info      # debug, info, warn, error
  format: text     # text, json

# Optional: configure import defaults
import:
  batch_size: 100
  content_field: text
  id_field: id
```

### Environment Variables

All configuration options can be set via environment variables. This is especially useful in Docker/Kubernetes environments.

| Variable | Equivalent Flag | Default | Description |
|----------|-----------------|---------|-------------|
| `CHROMA_HOST` | `--host` | `localhost` | ChromaDB server hostname |
| `CHROMA_PORT` | `--port` | `8000` | ChromaDB server port |
| `TENANT` or `CHROMA_TENANT` | `--tenant` | `default_tenant` | Tenant name |
| `DATABASE` or `CHROMA_DATABASE` | `--database` | `default_database` | Database name |
| `COLLECTION` or `CHROMA_COLLECTION` | `--collection` | (none) | Default collection |
| `LOG_LEVEL` | `--log-level` | `info` | Logging level |
| `LOG_FORMAT` | `--log-format` | `text` | Log format (text/json) |
| `CMDCHROMA_CONFIG` | `--config` | (none) | Path to config file |
| `NVIDIA_API_KEY` | (chat only) | (none) | Required for NVIDIA NIM models in `chroma chat` |

### CLI Flags (Recap)

All flags are also documented under **Global Flags** above. They override both environment variables and config file settings.

```bash
# Example: override config temporarily
chroma ping --host staging.example.com --port 9000
```

### Tips

- Set defaults in a config file to avoid typing flags repeatedly
- Use environment variables for CI/CD pipelines and Docker containers
- Use CLI flags for one-off commands that need special settings
- Run `chroma config init` to generate a starter config file (TODO)

---

## 🎨 CLI Design Principles

cmdChroma follows the [clig.dev](https://clig.dev) guidelines:

- **Clear help text**: Every command has a detailed description and usage examples.
- **Sensible defaults**: Works out of the box with local ChromaDB.
- **Actionable errors**: When something fails, you get hints on how to fix it.
- **Consistent flags**: Same flag names and environment variables across commands.
- **Progress feedback**: Long-running operations (imports) show status updates.
- **User-friendly output**: Emoji and formatting make results easy to scan.

---

## 🐛 Troubleshooting

### "connection failed"
- Is ChromaDB running? `docker ps`
- Check host/port: `chroma ping --host localhost --port 8000`
- Verify network connectivity

### "failed to initialize embedding engine"
- Ensure model files exist in `models/` directory
- Run setup script: `./.ci/scripts/setup.sh`
- Or specify paths manually: `--model-path`, `--tokenizer-path`, `--onnx-lib`

### "collection not found"
- List collections: `chroma collections`
- Create collection first: `chroma create <name>`

### LLM chat fails (`chroma chat`)

**For Ollama:**
- Is Ollama running? Start it: `ollama serve`
- Pull a model: `ollama pull qwen:0.5b`
- Check Ollama is at `http://localhost:11434`

**For NVIDIA NIM:**
- Ensure `NVIDIA_API_KEY` environment variable is set
- Verify the model name uses `nim://` prefix (e.g., `nim://mistralai/mistral-7b-instruct`)
- Check your internet connection and API key permissions
- Default endpoint is `https://api.nvidia.com/v1`; override with `--nim-url` if needed

---

## 🧪 Testing

See [TESTING.md](TESTING.md) for the complete testing guide, including:

- Running unit tests with `make test`
- Running integration tests with `make venom`
- Running tests by category (smoke, commands, configuration, scenarios)
- Adding new integration tests with the hierarchical organization

**Quick commands:**

```bash
# Run unit tests and build verification
make test

# Run all integration tests (requires ChromaDB running)
make venom

# Run full development cycle (fmt, lint, build, test, integration)
make dev

# Check code style
make lint
```

The integration test suite uses [Venom](https://github.com/ovh/venom) and is organized into feature-based directories under `.ci/tests/`. Each test file is tagged for category (smoke, slow, requires-server) to help you run subsets efficiently.

---

## 📜 License & Attribution

---

Licensed under the Apache License 2.0.

### Third-Party Attributions
- **ChromaDB**: The AI-native open-source embedding database
- **ONNX Runtime**: High-performance ML inferencing by Microsoft
- **Hugging Face**: `all-MiniLM-L6-v2` transformer model

---

## 🤝 Contributing

Contributions welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) (to be created) and follow the clig.dev guidelines for CLI changes.

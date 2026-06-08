# 🚀 cmdChroma CLI

A high-performance, user-friendly command-line tool for **ChromaDB** with local AI embeddings. Designed for developers building **RAG (Retrieval-Augmented Generation)** pipelines who want to keep their data and AI processing entirely local.

[![Go CI](https://github.com/DONAR-0/cmdChroma/actions/workflows/ci.yml/badge.svg)](https://github.com/DONAR-0/cmdChroma/actions/workflows/ci.yml)
[![Integration Tests](https://github.com/DONAR-0/cmdChroma/actions/workflows/integration.yml/badge.svg)](https://github.com/DONAR-0/cmdChroma/actions/workflows/integration.yml)
[![Release](https://github.com/DONAR-0/cmdChroma/actions/workflows/release.yml/badge.svg)](https://github.com/DONAR-0/cmdChroma/actions/workflows/release.yml)
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
# Pull pre-built image [TODO]
docker pull DONAR-0/cmdchroma:latest

# Run (with network access to ChromaDB)
docker run --rm --network host DONAR-0/cmdchroma:latest ping
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

#### `ping` — Test Connection
Verify connectivity to your ChromaDB instance.
```bash
chroma ping                    # Default localhost:8000
chroma ping --host 192.168.1.10 --port 8080
chroma ping --timeout 10       # Custom timeout
```

#### `create` — Create Collection
Create a new collection to store your documents.
```bash
chroma create my_collection
chroma create my_collection --tenant my_tenant --database my_db
```

#### `add` — Add Documents
Add one or more documents with automatic local embedding.
```bash
# Single document
chroma add my_collection --doc "The capital of France is Paris."

# Multiple documents (batch)
chroma add my_collection \
  --doc "Go is a compiled language." \
  --doc "Python is interpreted." \
  --doc "JavaScript runs in browsers."

# With custom IDs
chroma add my_collection \
  --doc "Document 1" --id "doc-1" \
  --doc "Document 2" --id "doc-2"

# From a file using xargs
cat docs.txt | xargs -I {} chroma add my_collection --doc "{}"
```

#### `query` — Search Documents
Perform semantic search with natural language queries.
```bash
# Single query
chroma query my_collection --query "What is Go programming?"

# Batch queries (much faster than running separately)
chroma query my_collection \
  --query "How to use vectors?" \
  --query "What is RAG?" \
  --query "Explain embeddings"

# Control number of results
chroma query my_collection --query "AI concepts" --n-results 10
```

#### `import` — Import JSONL File
Bulk import from JSONL files (e.g., Hugging Face datasets).
```bash
# Basic import (expects fields: 'text' and 'id')
chroma import my_collection data.jsonl

# Custom field mapping
chroma import my_collection data.jsonl \
  --field-content "content" \
  --field-id "uuid"

# Import with metadata and limit
chroma import my_collection data.jsonl \
  --field-metadata "author" \
  --field-metadata "category" \
  --all-metadata \
  --n-ingest 1000 \
  --batch-size 500
```


#### `chat` — RAG Q&A
Ask questions about your collection using AI-powered retrieval (supports Ollama and NVIDIA NIM).

**Ollama (local):**
```bash
# Requires Ollama running (default: localhost:11434)
chroma chat my_collection "What are the main topics?"

# Use a specific model
chroma chat my_collection "Summarize this" --llm-model llama2

# Retrieve more context for complex questions
chroma chat my_collection "Explain in detail" --n-results 10
```

**NVIDIA NIM (cloud API):**
```bash
# Set your NVIDIA API key
export NVIDIA_API_KEY="your-api-key-here"

# Use a NIM model (prefix with nim://)
# Check available models: curl -H "Authorization: Bearer $NVIDIA_API_KEY" https://integrate.api.nvidia.com/v1/models
chroma chat my_collection "What are the main topics?" --llm-model nim://mistralai/mistral-7b-instruct-v0.3

# Custom NIM endpoint (default: https://integrate.api.nvidia.com/v1)
chroma chat my_collection "Explain this" --llm-model nim://meta/llama-3.1-8b-instruct

# Filter irrelevant matches by distance (e.g., only accept results with distance <= 20)
chroma chat my_collection "Explain this" --llm-model nim://meta/llama-3.1-8b-instruct --distance-threshold 20
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

```bash
# Unit tests
go test ./...
# or
make test

# Integration tests
make venom

# Lint
make lint

# Code generation (build metadata)
go generate ./...
```

---

## 📜 License & Attribution

Licensed under the Apache License 2.0.

### Third-Party Attributions
- **ChromaDB**: The AI-native open-source embedding database
- **ONNX Runtime**: High-performance ML inferencing by Microsoft
- **Hugging Face**: `all-MiniLM-L6-v2` transformer model

---

## 🤝 Contributing

Contributions welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) (to be created) and follow the clig.dev guidelines for CLI changes.

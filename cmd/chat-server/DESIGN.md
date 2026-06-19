# Chat Server — Design Spec

**Date:** 2026-06-15
**Status:** Approved
**Author:** DONAR-0 / Claude

---

## Overview

A shared, team-use RAG chat server built around existing cmdChroma infrastructure.
Users connect via TUI or Web UI, start a session pinned to a knowledge base collection,
and get streaming LLM responses grounded in retrieved documents.

---

## Design Decisions

| # | Question | Chosen | Rationale |
|---|---|---|---|
| 1 | Audience | Team/shared server | Per-user sessions, collections per session, no full user auth |
| 2 | Conversation history | Session memory | Messages accumulate during session, cleared on restart or explicit clear |
| 3 | Interfaces | TUI + Web UI | REST API server; both clients connect via HTTP + SSE |
| 4 | Streaming | SSE | Token-by-token push; no WebSocket complexity needed |
| 5 | Auth | Static API key | One key shared by team, in `X-API-Key` header |
| 6 | Model selection | Per-request | Client passes model in request body; server forwards to LLM provider |
| 7 | Architecture | Layered monolith | One binary; handlers → service → integrations, cleanly separated |
| 8 | Collections | Per-session | Client picks a collection when starting a session; collections pre-loaded |

---

## Architecture

```
                    ┌─────────────────────────────────────┐
                    │           HTTP Server (Gin)          │
                    │  middleware: API key, logging, panic│
                    └──────────────┬──────────────────────┘
                                   │
                    ┌──────────────▼──────────────────────┐
                    │           API Router                  │
                    │  POST /api/chat      (streaming SSE) │
                    │  GET  /api/sessions                  │
                    │  DELETE /api/sessions/:id            │
                    │  POST /api/query                     │
                    │  GET  /api/collections               │
                    │  GET  /health                        │
                    └──────────────┬──────────────────────┘
                                   │
         ┌─────────────────────────┼─────────────────────────┐
         │                         │                         │
  ┌──────▼──────┐         ┌───────▼───────┐        ┌───────▼───────┐
  │ chat_service│         │ session_store │        │ collection_mgr│
  │             │         │ (in-memory)   │        │ (in-memory)   │
  │ query docs  │         │ sessions[]    │        │ collections   │
  │ build prompt│         │ messages[]    │        │ from config   │
  │ call LLM    │         └───────────────┘        └───────────────┘
  │ stream SSE  │                                                  │
  └──────┬──────┘                                                  │
         │                                                         │
  ┌──────▼──────────────────────────────────────┐                │
  │           Integration Layer                  │                │
  │  ChromaDB client (internal/client)       │                │
  │  Ollama / NIM providers (internal/llm)  │                │
  │  ONNX embedder (internal/onnx)            │                │
  └────────────────────────────────────────────┘                │
                                                              │
                                                    ┌──────────▼──────────┐
                                                    │ pre-loaded collections│
                                                    │ (config-driven)       │
                                                    └───────────────────────┘
```

### Session Lifecycle

1. Client POSTs to `/api/chat` with `{ collection, model, message, session_id? }`
2. If `session_id` not provided, a new UUID is generated
3. Session is looked up (or created) in `memory_session_store`
4. Chat history for that session is retrieved
5. RAG pipeline fires: query ChromaDB → build prompt with history → stream LLM response via SSE
6. Every message (user + assistant) is appended to the session's message list
7. On `DELETE /api/sessions/:id`, session is cleared; messages removed, session ID preserved (for reconnect)

---

## API Specification

### Common Headers

All `/api/*` requests require:
```
X-API-Key: <configured_api_key>
Content-Type: application/json
```

### `GET /health`

Returns server health. No auth required.

Response:
```json
{ "status": "ok", "chroma": "connected", "embedder": "ready" }
```

### `GET /api/collections`

List available collections (loaded from server config).

Response:
```json
{
  "collections": [
    { "name": "tech_faq", "description": "Tech FAQ knowledge base", "dimension": 384 },
    { "name": "articles", "description": "10K article corpus", "dimension": 384 }
  ]
}
```

### `POST /api/chat`

Streaming chat with RAG context.

**Request body:**
```json
{
  "collection": "tech_faq",
  "model": "nim://google/gemma-2-2b-it",
  "message": "What is Python?",
  "session_id": "optional-uuid-here",
  "n_results": 3,
  "distance_threshold": 0
}
```

**Response:** `Content-Type: text/event-stream`

Each event:
```
event: token
data: {"content": " Py"}

event: token
data: {"content": "thon"}

event: done
data: {"total_tokens": 142}
```

**Error events:**
```
event: error
data: {"error": "collection not found"}
```

### `GET /api/sessions`

List active sessions for this API key.

Response:
```json
{
  "sessions": [
    { "id": "uuid-1", "collection": "articles", "message_count": 6, "created_at": "..." },
    { "id": "uuid-2", "collection": "tech_faq", "message_count": 2, "created_at": "..." }
  ]
}
```

### `DELETE /api/sessions/:id`

Clear a session's message history.

Response:
```json
{ "status": "cleared", "session_id": "uuid-1" }
```

### `POST /api/query`

Non-streaming semantic query — returns raw retrieval results.

**Request body:**
```json
{
  "collection": "tech_faq",
  "query": "What is Python?",
  "n_results": 3,
  "distance_threshold": 0
}
```

**Response:**
```json
{
  "documents": [
    { "id": "rag_001", "content": "...", "distance": 0.1842, "metadata": {...} },
    { "id": "rag_005", "content": "...", "distance": 0.3421, "metadata": {...} }
  ]
}
```

---

## Configuration

All server configuration via YAML file (`chatting_server.yaml`) or environment variables:

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  api_key: "changeme"           # X-API-Key required
  cors_allowed_origins:
    - "http://localhost:3000"

chroma:
  url: "http://localhost:8000"
  tenant: "default_tenant"
  database: "default_database"

embedder:
  model_path: "./models/all-MiniLM-L6-v2/model.onnx"

llm:
  default_model: "qwen:0.5b"
  # override defaults per provider:
  ollama_url: "http://localhost:11434"
  nim_url: "https://integrate.api.nvidia.com/v1"

collections:
  - name: "tech_faq"
    description: "Tech FAQ knowledge base"
  - name: "articles"
    description: "10K article corpus"
  - name: "medical_qa"
    description: "Medical Q&A corpus"
```

Environment variable overrides (prefix `CHAT`):
```
CHAT_SERVER_API_KEY=...
CHAT_SERVER_PORT=8080
CHAT_CHROMA_URL=http://localhost:8000
CHAT_OLLAMA_URL=http://localhost:11434
CHAT_NIM_URL=https://integrate.api.nvidia.com/v1
CHAT_LLM_DEFAULT_MODEL=qwen:0.5b
```

---

## Session Store (In-Memory)

```go
type Session struct {
    ID         string
    APIKey     string
    Collection string
    Messages   []Message   // append-only per turn
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

type Message struct {
    Role    string    // "user" or "assistant"
    Content string
    Tokens  int       // assistant only
}

type SessionStore struct {
    mu        sync.RWMutex
    sessions  map[string]*Session  // key: sessionID
}
```

Sessions are stored in memory and are lost on server restart. The `DELETE` endpoint clears the message list but preserves the session entry so concurrent reconnect works.

---

## Dependencies

| Package | Purpose |
|---|---|
| `github.com/gin-gonic/gin` | HTTP framework |
| `github.com/charmbracelet/bubbletea` | TUI client |
| `github.com/google/uuid` | Session ID generation |
| `gopkg.in/yaml.v3` | Config parsing |
| `github.com/DONAR-0/cmdChroma/internal/client` | ChromaDB client (imported) |
| `github.com/DONAR-0/cmdChroma/internal/llm` | LLM providers (imported) |
| `github.com/DONAR-0/cmdChroma/internal/onnx` | ONNX embedder (imported) |

---

## TUI Client (Bubble Tea)

First client to implement. Key screens:

```
┌─────────────────────────────────────┐
│ chatting_server                    │
│ [model: gemma-2-2b-it] [col: tech]  │
├─────────────────────────────────────┤
│                                     │
│ [user] What is Python?              │
│                                     │
│ [assistant] Python is a high-level  │
│ programming language known for      │
│ its readability...                  │
│                                     │
├─────────────────────────────────────┤
│ > Type your message...              │
│                                     │
└─────────────────────────────────────┘
```

Features:
- SSE connection via HTTP client
- Session ID persisted in `~/.config/chatting_server/session`
- Model + collection selector (dropdown or flag)
- Syntax-highlighted markdown in assistant messages (via `lipgloss`)
- `Ctrl+C` exits, `Ctrl+L` clears screen

---

## Web UI (future)

React or Vue SPA. Not in initial scope — API is the contract.

---

## Acceptance Criteria

- [ ] Server starts, connects to ChromaDB, loads embedder
- [ ] `/health` returns status without API key
- [ ] All `/api/*` routes reject requests without valid `X-API-Key`
- [ ] `POST /api/chat` streams tokens via SSE formatted as `event: token\ndata: {...}`
- [ ] Session message history is maintained across multiple messages in same session
- [ ] `DELETE /api/sessions/:id` clears message history
- [ ] Non-existent collection returns `404` with descriptive error
- [ ] TUI connects to server, renders streaming response, respects session continuity
- [ ] Config file loaded from `chatting_server.yaml` in working directory
- [ ] Env vars with `CHAT_` prefix override config file values

---

## What Is In Scope (v1)

- Single server binary
- TUI client
- In-memory sessions
- SSE streaming
- All LLM providers supported by cmdChroma
- Config via YAML + env vars

## What Is Out of Scope (v1)

- Web UI (future iteration)
- Persistent session storage (database)
- User accounts / multiple API keys
- Multiple simultaneous LLM calls per session (queue or reject)
- TLS (future: behind reverse proxy)
- Rate limiting
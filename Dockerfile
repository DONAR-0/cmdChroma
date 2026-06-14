# Stage 1: Build binary with dependencies
FROM golang:1.25.7-bookworm AS builder

# Install build dependencies for CGO and ONNX
RUN apt-get update && apt-get install -y \
    gcc \
    g++ \
    make \
    curl \
    tar \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Download models and structure paths clearly
RUN mkdir -p models/all-MiniLM-L6-v2 && \
    mkdir -p models/onnx_runtime/lib && \
    curl -L "https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2/resolve/main/onnx/model.onnx" \
      -o models/all-MiniLM-L6-v2/model.onnx && \
    curl -L "https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2/resolve/main/tokenizer.json" \
      -o models/all-MiniLM-L6-v2/tokenizer.json && \
    curl -L "https://github.com/microsoft/onnxruntime/releases/download/v1.24.2/onnxruntime-linux-x64-1.24.2.tgz" \
      -o onnxruntime.tgz && \
    tar -xzf onnxruntime.tgz -C /tmp && \
    cp /tmp/onnxruntime-linux-x64-*/lib/libonnxruntime.so.1.24.2 models/onnx_runtime/lib/libonnxruntime.so.1 && \
    # FIX: Added '-sf' to force overwrite the symlink if it was copied from your host machine
    ln -sf libonnxruntime.so.1 models/onnx_runtime/lib/libonnxruntime.so && \
    rm -rf onnxruntime.tgz /tmp/onnxruntime-linux-x64-*

# Build the binary using explicit host linker RPATH configs to prevent Go's $ORIGIN truncation bugs
RUN CGO_ENABLED=1 \
    GOOS=linux \
    GOARCH=amd64 \
    CGO_LDFLAGS="-L/build/tokenizerLib -L/build/models/onnx_runtime/lib -ltokenizers -lstdc++ -Wl,-rpath='\$ORIGIN/../models/tokenizerLib:\$ORIGIN/../models/onnx_runtime/lib'" \
    go build -o cmdChroma ./cmd/chroma


# Stage 2: Runtime image (Bookworm-slim matches builder GLIBC)
FROM debian:bookworm-slim

# Install runtime dependencies for ONNX and tokenizers
RUN apt-get update && apt-get install -y \
    libgomp1 \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Rebuild expected layout
RUN mkdir -p /app/bin /app/models

# Copy assets cleanly from builder
COPY --from=builder /build/cmdChroma /app/bin/cmdChroma
COPY --from=builder /build/models/ /app/models/
COPY --from=builder /build/tokenizerLib/ /app/models/tokenizerLib/

# Secure non-root context
RUN useradd -m -u 1000 cmdchroma && \
    chown -R cmdchroma:cmdchroma /app
USER cmdchroma

EXPOSE 8000

# Health check validation
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD /app/bin/cmdChroma --version || exit 1

ENTRYPOINT ["/app/bin/cmdChroma"]
CMD ["--help"]

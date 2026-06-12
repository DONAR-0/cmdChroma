# Multi-stage build for cmdChroma
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

# Download models (same as .ci/scripts/setup.sh)
RUN mkdir -p models/all-MiniLM-L6-v2 && \
    mkdir -p models/onnx_runtime/lib && \
    curl -L "https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2/resolve/main/onnx/model.onnx" \
      -o models/all-MiniLM-L6-v2/model.onnx && \
    curl -L "https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2/resolve/main/tokenizer.json" \
      -o models/all-MiniLM-L6-v2/tokenizer.json && \
    curl -L "https://github.com/microsoft/onnxruntime/releases/download/v1.24.2/onnxruntime-linux-x64-1.24.2.tgz" \
      -o onnxruntime.tgz && \
    tar -xzf onnxruntime.tgz -C models/onnx_runtime --strip-components=1 && \
    rm onnxruntime.tgz

# Build the binary with RPATH support and tokenizers linking
# - RPATH: $ORIGIN/../models/onnx_runtime/lib:$ORIGIN/../models/tokenizerLib
#   This expects the binary to be located in a subdirectory of the app root (e.g., /app/bin)
#   so that ../models points to the models directory.
RUN CGO_ENABLED=1 \
    GOOS=linux \
    GOARCH=amd64 \
    CGO_LDFLAGS="-L$(pwd)/tokenizerLib -ltokenizers -lstdc++" \
    go build -ldflags="-r '$$ORIGIN/../models/onnx_runtime/lib:$$ORIGIN/../models/tokenizerLib'" \
    -o cmdChroma ./cmd/chroma

# Stage 2: Runtime image
FROM debian:bullseye-slim

# Install runtime dependencies for ONNX and tokenizers
RUN apt-get update && apt-get install -y \
    libgomp1 \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Create bin directory and copy binary from builder
RUN mkdir -p /app/bin
COPY --from=builder /build/cmdChroma /app/bin/cmdChroma

# Copy models (ONNX model, tokenizer, ONNX runtime library)
COPY --from=builder /build/models/ /app/models/

# Create symlink for ONNX runtime library compatibility (some distros have .so.1)
RUN ln -s /app/models/onnx_runtime/lib/libonnxruntime.so.1 /app/models/onnx_runtime/lib/libonnxruntime.so || true

# Create non-root user for security
RUN useradd -m -u 1000 cmdchroma && \
    chown -R cmdchroma:cmdchroma /app
USER cmdchroma

# Expose port for Chroma server (when used as server)
EXPOSE 8000

# Health check (verify binary runs)
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD /app/bin/cmdChroma --version || exit 1

# Default entrypoint
ENTRYPOINT ["/app/bin/cmdChroma"]
CMD ["--help"]
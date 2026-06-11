#!/usr/bin/env bash

set -e

# Keep this script runnable from anywhere by resolving the repo root
# Works in pure POSIX shell (does not rely on bash-only variables).
REPO_ROOT=$(cd "$(dirname "$0")/../.." && pwd)

# Configuration
MODEL_DIR="$REPO_ROOT/models"
MINILM_URL="https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2/resolve/main/onnx/model.onnx"
TOKENIZER_URL="https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2/resolve/main/tokenizer.json"
ONNX_URL="https://github.com/microsoft/onnxruntime/releases/download/v1.24.2/onnxruntime-linux-x64-1.24.2.tgz"

echo "📂 Creating directory structure..."
mkdir -p $MODEL_DIR/all-MiniLM-L6-v2
mkdir -p $MODEL_DIR/onnx_runtime

echo "🤖 Downloading all-MiniLM-L6-v2 ONNX model..."
if [ ! -f "$MODEL_DIR/all-MiniLM-L6-v2/model.onnx" ]; then
  curl -L "$MINILM_URL" -o "$MODEL_DIR/all-MiniLM-L6-v2/model.onnx"
else
  echo "✅ Model already exists: $MODEL_DIR/all-MiniLM-L6-v2/model.onnx"
fi

if [ ! -f "$MODEL_DIR/all-MiniLM-L6-v2/tokenizer.json" ]; then
  curl -L "$TOKENIZER_URL" -o "$MODEL_DIR/all-MiniLM-L6-v2/tokenizer.json"
else
  echo "✅ Tokenizer already exists: $MODEL_DIR/all-MiniLM-L6-v2/tokenizer.json"
fi

echo "📦 Downloading ONNX Runtime (Linux x64)..."
if [ ! -f "$MODEL_DIR/onnx_runtime/lib/libonnxruntime.so.1" ]; then
  curl -L "$ONNX_URL" -o onnxruntime.tgz
  tar -xzf onnxruntime.tgz -C "$MODEL_DIR/onnx_runtime" --strip-components=1
  rm onnxruntime.tgz
else
  echo "✅ ONNX Runtime already exists: $MODEL_DIR/onnx_runtime/lib/libonnxruntime.so.1"
fi

echo "✅ Setup complete! Models and libraries are in $MODEL_DIR"

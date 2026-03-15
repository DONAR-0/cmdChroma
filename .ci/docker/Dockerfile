# syntax=docker/dockerfile:1.4

# --- build stage -------------------------------------------------------------
FROM golang:1.22-bullseye AS builder

WORKDIR /src

# Cache Go module downloads between builds
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source tree
COPY . .

# Build the CLI binary
# NOTE: This builds a linux/amd64 binary by default.
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go build -o /out/cmdChroma ./cmd/chroma

# --- runtime stage -----------------------------------------------------------
FROM gcr.io/distroless/base-debian11

COPY --from=builder /out/cmdChroma /usr/local/bin/cmdChroma

ENTRYPOINT ["/usr/local/bin/cmdChroma"]
CMD ["--help"]

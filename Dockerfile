# Multi-stage build for namazu
# Stage 1: Build frontend
# Stage 2: Build Go binary with embedded static files
# Stage 3: Minimal runtime image

# =============================================================================
# Stage 1: Build Frontend
# =============================================================================
FROM node:22-alpine AS frontend-builder

# Install pnpm
RUN corepack enable && corepack prepare pnpm@latest --activate

WORKDIR /app

# Copy workspace and lock files for dependency caching
COPY pnpm-workspace.yaml pnpm-lock.yaml package.json* ./
COPY web/package.json ./web/

# Install dependencies
RUN pnpm install --frozen-lockfile

# Copy frontend source files
COPY web/ ./web/

# Build frontend (outputs to ../cmd/namazu/static per vite.config.ts)
WORKDIR /app/web
RUN pnpm build

# =============================================================================
# Stage 2: Build Go Binary
# =============================================================================
FROM golang:1.24-alpine AS go-builder

# Build argument for commit hash (passed from CI)
ARG COMMIT_HASH=unknown

# Install git and ca-certificates for Go modules and HTTPS
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy go mod files for dependency caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Copy frontend build output to embed location
COPY --from=frontend-builder /app/cmd/namazu/static ./cmd/namazu/static

# Build the binary
# CGO_ENABLED=0 for static binary
# -ldflags="-s -w" to strip debug info and reduce size
# -X to embed commit hash for version tracking
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X github.com/otiai10/namazu/internal/version.CommitHash=${COMMIT_HASH}" \
    -o /app/namazu \
    ./cmd/namazu

# =============================================================================
# Stage 3: Runtime Image
# =============================================================================
FROM alpine:3.19 AS runtime

# Install ca-certificates for HTTPS and tzdata for timezone
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user for security
RUN addgroup -g 1000 namazu && \
    adduser -u 1000 -G namazu -s /bin/sh -D namazu

WORKDIR /app

# Copy binary from builder
COPY --from=go-builder /app/namazu /app/namazu

# Set ownership
RUN chown -R namazu:namazu /app

# Switch to non-root user
USER namazu

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
ENTRYPOINT ["/app/namazu"]

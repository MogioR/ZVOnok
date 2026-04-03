# ────────────────────────────────────────────────────────────────────────────
# Stage 1 — Build Vue frontend
# ────────────────────────────────────────────────────────────────────────────
FROM node:20-alpine AS frontend-builder

WORKDIR /frontend

# Install deps first (separate layer → cached unless package.json changes)
COPY frontend/package*.json ./
RUN npm install --silent

COPY frontend/ ./

# Build and output straight to /dist (avoids cross-directory path in vite.config)
RUN npm run build -- --outDir /dist --emptyOutDir

# ────────────────────────────────────────────────────────────────────────────
# Stage 2 — Build Go backend
# ────────────────────────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS go-builder

WORKDIR /build

# Copy source first so go mod tidy can resolve imports
COPY backend/ ./
COPY --from=frontend-builder /dist ./static

# Resolve deps and generate go.sum, then build
RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o zvonok .

# ────────────────────────────────────────────────────────────────────────────
# Stage 3 — Minimal runtime image (~10 MB)
# ────────────────────────────────────────────────────────────────────────────
FROM alpine:3.19

# Runtime deps: ca-certs, tz, ffmpeg, python+pip for yt-dlp
RUN apk add --no-cache ca-certificates tzdata ffmpeg python3 py3-pip nodejs && \
    python3 -m pip install --no-cache-dir --break-system-packages yt-dlp

WORKDIR /app

COPY --from=go-builder /build/zvonok  ./zvonok
COPY --from=go-builder /build/static  ./static

# Non-root user with a home directory so yt-dlp cache works
RUN addgroup -S zvonok && adduser -S -h /home/zvonok -G zvonok zvonok && \
    mkdir -p /home/zvonok && chown zvonok:zvonok /home/zvonok
USER zvonok

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:${PORT:-8080}/ > /dev/null || exit 1

CMD ["./zvonok"]

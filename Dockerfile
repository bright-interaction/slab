# Stage 1: Build SvelteKit SPA
FROM oven/bun:1-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/bun.lock ./
RUN bun install --frozen-lockfile
COPY frontend/ .
RUN bun run build

# Stage 2: Build Go binary
FROM golang:1.26-alpine AS backend
WORKDIR /app
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/frontend/build ./cmd/server/frontend/build
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/server

# Stage 3: Production image
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=oven/bun:1-alpine /usr/local/bin/bun /usr/local/bin/bun
RUN addgroup -S atomicsite && adduser -S atomicsite -G atomicsite
RUN mkdir -p /app/data && chown -R atomicsite:atomicsite /app
COPY --from=backend /server /app/server
WORKDIR /app
USER atomicsite
ENV DATA_DIR=/app/data
EXPOSE 8080
ENTRYPOINT ["/app/server"]

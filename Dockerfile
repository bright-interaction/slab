# Build context: this Dockerfile is built from the parent `automations/`
# directory so it can COPY both `atomicsite/` (this app) and the sibling
# `CookieProof/` source for the embedded widget bundle. the CI system CI is
# configured accordingly; locally:
#
#   cd automations && docker build -f atomicsite/Dockerfile -t atomicsite .

# Stage 1: Build the CookieProof embed widget bundle. This produces the same
# JS that gets vendored into the Go binary via go:embed (see
# atomicsite/internal/builder/widget_embed.go).
FROM oven/bun:1-alpine AS widget
WORKDIR /app
COPY CookieProof/package.json CookieProof/bun.lock ./
RUN --mount=type=cache,target=/root/.bun/install/cache \
    bun install --frozen-lockfile
COPY CookieProof/ .
RUN bun run build

# Stage 2: Build atomicsite's SvelteKit admin SPA.
FROM oven/bun:1-alpine AS frontend
WORKDIR /app/frontend
COPY atomicsite/frontend/package.json atomicsite/frontend/bun.lock ./
RUN --mount=type=cache,target=/root/.bun/install/cache \
    bun install --frozen-lockfile
COPY atomicsite/frontend/ .
RUN bun run build

# Stage 3: Build the Go server. Pulls in the freshly-built widget bundle so
# go:embed picks up the latest bytes at compile time.
FROM golang:1.26-alpine AS backend
WORKDIR /app
RUN apk add --no-cache git ca-certificates
COPY atomicsite/go.mod atomicsite/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download
COPY atomicsite/ .
COPY --from=frontend /app/frontend/build ./cmd/server/frontend/build
COPY --from=widget /app/dist/cookieproof.embed.esm.js ./internal/builder/assets/cookieproof.embed.esm.js
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/server

# Stage 4: Production image.
FROM alpine:3.20
# bun is dynamically linked against libstdc++ / libgcc / unwind; without these
# `bun install` inside the runtime aborts with "_ZNSt... symbol not found".
# chromium is used by the /api/agent/screenshot endpoint (chromedp) so the
# agent can see the rendered output and iterate against pixels.
# nss / freetype / harfbuzz / ca-certificates are chromium runtime deps.
RUN apk add --no-cache ca-certificates tzdata libstdc++ libgcc \
    chromium nss freetype harfbuzz ttf-freefont
ENV CHROMEDP_HEADLESS_FLAGS="" \
    CHROMEDP_NO_SANDBOX=1
COPY --from=oven/bun:1-alpine /usr/local/bin/bun /usr/local/bin/bun
RUN addgroup -S atomicsite && adduser -S atomicsite -G atomicsite
RUN mkdir -p /app/data && chown -R atomicsite:atomicsite /app
COPY --from=backend /server /app/server
WORKDIR /app
USER atomicsite
ENV DATA_DIR=/app/data
EXPOSE 8080
ENTRYPOINT ["/app/server"]

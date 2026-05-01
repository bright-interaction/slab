# Atomic Site Dockerfile (self-contained).
#
# Builds entirely from the atomicsite/ directory. The CookieProof widget
# bundle is vendored at internal/builder/assets/cookieproof.embed.esm.js
# and embedded into the Go binary via go:embed (see widget_embed.go), so
# Atomic Site has zero runtime or build-time dependency on the sibling
# CookieProof source. To refresh the widget bundle from source, run a
# CookieProof build separately and commit the regenerated asset.
#
# Local build:
#   cd atomicsite && docker build -f Dockerfile -t atomicsite .

# Stage 1: SvelteKit admin SPA.
FROM oven/bun:1-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/bun.lock ./
RUN --mount=type=cache,target=/root/.bun/install/cache \
    bun install --frozen-lockfile
COPY frontend/ .
RUN bun run build

# Stage 2: Go server. The frontend build is copied in for go:embed.
#
# CGO note: the analyticsdb package uses marcboeker/go-duckdb/v2 which
# requires CGO + libduckdb. We build with CGO_ENABLED=1, target musl, and
# install build-base + gcc + g++ + libstdc++. The resulting binary is
# dynamically linked against libstdc++/libgcc; both are present in the
# runtime image (chromium runtime needs them too) so no extra runtime
# libs are required.
FROM golang:1.26-alpine AS backend
WORKDIR /app
RUN apk add --no-cache git ca-certificates build-base gcc g++ musl-dev libstdc++
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download
COPY . .
COPY --from=frontend /app/frontend/build ./cmd/server/frontend/build
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/server

# Stage 3: Production image.
FROM alpine:3.20
# bun + chromium + DuckDB (via go-duckdb) are dynamically linked against
# libstdc++ / libgcc / unwind; without these the binary won't start.
# nss / freetype / harfbuzz / ca-certificates are chromium runtime deps.
# The DuckDB sqlite_scanner extension downloads on first `LOAD sqlite`
# (~6 MB); after that it's cached at /home/atomicsite/.duckdb.
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

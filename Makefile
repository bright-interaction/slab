.PHONY: dev frontend frontend-dev frontend-install build build-server vet sqlc clean

# Run Go server in dev mode (frontend served separately via `make frontend-dev`)
dev:
	go run ./cmd/server

# Build SvelteKit frontend and sync into the embed directory
frontend:
	cd frontend && bun run build
	rm -rf cmd/server/frontend/build
	mkdir -p cmd/server/frontend
	cp -r frontend/build cmd/server/frontend/build

frontend-install:
	cd frontend && bun install

frontend-dev:
	cd frontend && bun run dev

# Full production build: frontend then Go binary with embedded SPA
build: frontend build-server

build-server:
	go build -o bin/atomicsite ./cmd/server

vet:
	go vet ./...

sqlc:
	sqlc generate

clean:
	rm -rf bin/ cmd/server/frontend/build/
	cd frontend && rm -rf build/ .svelte-kit/ node_modules/

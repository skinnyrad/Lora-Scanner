.PHONY: web build dev test clean

BIN := lora-scanner

web:
	cd web && bun install --frozen-lockfile && bun run build
	touch web/dist/.gitkeep

# The web tests fail the build if the frontend references anything it doesn't embed (it must run offline).
build: web
	go test ./web
	go build -o $(BIN) .

test:
	go vet ./... && go test -race ./...

# Run `make dev` for the Go API; in another shell run `cd web && bun run dev`
# (the Bun dev server proxies /api to :5000).
dev:
	go run .

clean:
	rm -rf $(BIN) web/dist/*
	touch web/dist/.gitkeep

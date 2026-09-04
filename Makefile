GO_PACKAGES := ./cmd/... ./internal/...

.PHONY: build build-go build-web check check-packaging dev-agent dev-init dev-server dev-web fmt fmt-check generate lint release security setup test typecheck

setup:
	npm --prefix web install

generate:
	go tool sqlc generate
	go tool oapi-codegen --config api/public.cfg.yaml api/public.openapi.yaml
	go tool oapi-codegen --config api/agent.cfg.yaml api/agent.openapi.yaml
	npm --prefix web run generate:api

fmt:
	gofmt -w $$(find cmd internal -name '*.go' -type f)

fmt-check:
	@files="$$(gofmt -l $$(find cmd internal -name '*.go' -type f))"; \
	if [ -n "$$files" ]; then echo "Go files need formatting:"; echo "$$files"; exit 1; fi

lint:
	go vet $(GO_PACKAGES)
	npm --prefix web run lint

typecheck:
	npm --prefix web run typecheck

test:
	go test $(GO_PACKAGES)
	npm --prefix web run test

build-go:
	mkdir -p bin
	go build -o bin/webycp-server ./cmd/webycp-server
	go build -o bin/webycp-agent ./cmd/webycp-agent

build-web:
	npm --prefix web run build

build: build-go build-web

check-packaging:
	sh -n packaging/ubuntu/install.sh
	sh -n packaging/ubuntu/upgrade.sh
	sh -n scripts/release.sh
	sh -n scripts/security.sh

check: check-packaging fmt-check lint typecheck test build

security:
	./scripts/security.sh

release:
	@test -n "$(VERSION)" || { echo "VERSION is required, for example: make release VERSION=1.0.0"; exit 1; }
	./scripts/release.sh "$(VERSION)" $(RELEASE_FLAGS)

dev-server:
	WEBYCP_DATABASE_PATH=var/dev.db WEBYCP_AGENT_SOCKET=/tmp/webycp-agent.sock WEBYCP_SECURE_COOKIE=false go run ./cmd/webycp-server

dev-init:
	WEBYCP_DATABASE_PATH=var/dev.db go run ./cmd/webycp-server admin init

dev-agent:
	WEBYCP_AGENT_SOCKET=/tmp/webycp-agent.sock go run ./cmd/webycp-agent

dev-web:
	npm --prefix web run dev

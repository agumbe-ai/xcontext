.PHONY: build run test test-race fmt vet docker-build validate-plugins
build:
	go build -o bin/xcontext ./services/api/cmd/xcontext
run:
	go run ./services/api/cmd/xcontext
test:
	go test ./...
test-race:
	go test -race ./...
fmt:
	gofmt -w services
vet:
	go vet ./...
docker-build:
	docker build -t xcontext:local .
validate-plugins:
	./scripts/validate-claude-plugin.sh
	./scripts/validate-cursor-plugin.sh

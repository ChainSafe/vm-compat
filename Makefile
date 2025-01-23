GOLANGCI := $(GOPATH)/bin/golangci-lint

.PHONY: build-analyzer
build-analyser:
	go build -o ./bin/analyser ./cmd/analyser/main.go

.PHONY: get
get:
	go mod download && go mod tidy

.PHONY: get_lint
get_lint:
	@if [ ! -f ./bin/golangci-lint ]; then \
		curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.57.2; \
	fi;

.PHONY: lint
lint: get_lint
	@echo "  >  \033[32mRunning lint...\033[0m "
	./bin/golangci-lint run --config=./.golangci.yaml --fix

.PHONY: test
test:
	@echo "  >  \033[32mRunning sprinter-api tests...\033[0m "
	go test -v ./...
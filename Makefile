GOLANGCI := $(GOPATH)/bin/golangci-lint

.PHONY: analyzer
analyzer:
	go build -o ./bin/vm-compact ./main.go

.PHONY: get
get:
	go mod download && go mod tidy

.PHONY: get_lint
get_lint:
	@if [ ! -f ./bin/golangci-lint ]; then \
		curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.63.0; \
	fi;

.PHONY: lint
lint: get_lint
	@echo "  >  \033[32mRunning lint...\033[0m "
	./bin/golangci-lint run --config=./.golangci.yml --fix

.PHONY: test
test:
	@echo "  >  \033[32mRunning sprinter-api tests...\033[0m "
	go test -v ./...

# Run e2e tests
.PHONY: e2e-test
e2e-test: 
	go test ./e2e_tests -tags=integration -v

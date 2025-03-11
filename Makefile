GOLANGCI := $(GOPATH)/bin/golangci-lint
APP_NAME := analyzer
BUILD_DIR := bin
PLATFORMS := \
	linux/amd64 \
	linux/arm64 \
	darwin/amd64 \
	darwin/arm64 \
	windows/amd64
BINARIES := $(foreach platform,$(PLATFORMS),$(BUILD_DIR)/$(APP_NAME)-$(subst /,-,$(platform)))

.PHONY: analyzer
analyzer:
	go build -o ./bin/analyzer ./main.go

.PHONY: build-all
build-all: $(BINARIES)

$(BUILD_DIR)/$(APP_NAME)-%:
	@mkdir -p $(BUILD_DIR)
	@GOOS=$(word 1,$(subst -, ,$*)) GOARCH=$(word 2,$(subst -, ,$*)) go build -o $@ ./main.go
	@echo "Built $@"

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
version :=  $(shell cat version.txt)
plugin_name := fritzbox-telegraf-plugin

MAKEFLAGS += --no-print-directory
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

LDFLAGS := $(LDFLAGS) -X main.version=$(version) -X main.goos=$(GOOS) -X main.goarch=$(GOARCH)

.PHONY: all
all:
	@$(MAKE) deps
	@$(MAKE) $(plugin_name)

.PHONY: deps
deps:
	go mod download -x

.PHONY: $(plugin_name)
$(plugin_name):
	go build -ldflags "$(LDFLAGS)" -o .bin/$(plugin_name) ./cmd/$(plugin_name)

.PHONY: test
test:
	go test -covermode=atomic -coverprofile=coverage.out ./...

.PHONY: tidy
tidy:
	go mod verify
	go mod tidy

.PHONY: clean
clean:
	rm -rf .bin
	rm -rf .go
	rm -f *.out

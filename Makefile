version :=  $(shell cat version.txt)
plugin_name := fritzbox-telegraf-plugin
plugin_conf := fritzbox.conf

MAKEFLAGS += --no-print-directory
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

LDFLAGS := $(LDFLAGS) -X main.version=$(version) -X main.goos=$(GOOS) -X main.goarch=$(GOARCH)

.PHONY: all
all:
	@$(MAKE) deps
	@$(MAKE) $(plugin_name)
	@$(MAKE) dist

.PHONY: deps
deps:
	go mod download -x

.PHONY: $(plugin_name)
$(plugin_name):
	go build -ldflags "$(LDFLAGS)" -o .build/bin/$(plugin_name) ./cmd/$(plugin_name)
	cp $(plugin_conf) .build/bin/

.PHONY: dist
dist: all
	mkdir -p .build/dist
	tar czvf .build/dist/$(plugin_name)-$(GOOS)-$(GOARCH)-$(version).tar.gz -C .build/bin .
	zip -j .build/dist/$(plugin_name)-$(GOOS)-$(GOARCH)-$(version).zip .build/bin/*

.PHONY: test
test:
	go test -covermode=atomic -coverprofile=coverage.out ./...

.PHONY: tidy
tidy:
	go mod verify
	go mod tidy

.PHONY: clean
clean:
	rm -rf .build
	rm -rf .go
	rm -f *.out

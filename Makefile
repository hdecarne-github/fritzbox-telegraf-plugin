version :=  $(shell cat version.txt)
plugin_package := github.com/hdecarne/fritzbox-telegraf-plugin/plugins/inputs/fritzbox
plugin_name := fritzbox-telegraf-plugin
plugin_conf := fritzbox.conf

MAKEFLAGS += --no-print-directory
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

LDFLAGS := $(LDFLAGS) -X $(plugin_package).plugin=$(plugin_name) -X $(plugin_package).version=$(version) -X $(plugin_package).goos=$(GOOS) -X $(plugin_package).goarch=$(GOARCH)

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
ifneq (windows, $(GOOS))
	go build -ldflags "$(LDFLAGS)" -o build/bin/$(plugin_name) ./cmd/$(plugin_name)
else
	go build -ldflags "$(LDFLAGS)" -o build/bin/$(plugin_name).exe ./cmd/$(plugin_name)
endif
	cp $(plugin_conf) build/bin/

.PHONY: dist
dist:
	mkdir -p build/dist
	tar czvf build/dist/$(plugin_name)-$(GOOS)-$(GOARCH)-$(version).tar.gz -C build/bin .
ifneq (, $(shell command -v zip 2>/dev/null))
	zip -j build/dist/$(plugin_name)-$(GOOS)-$(GOARCH)-$(version).zip build/bin/*
else ifneq (, $(shell command -v 7z 2>/dev/null))
	7z a -bd build/dist/$(plugin_name)-$(GOOS)-$(GOARCH)-$(version).zip ./build/bin/*
endif

.PHONY: test
test:
	go test -covermode=atomic -coverprofile=coverage.out ./...

.PHONY: tidy
tidy:
	go mod verify
	go mod tidy

.PHONY: clean
clean:
	rm -rf build
	rm -rf .go
	rm -f *.out

plugin_name := $(shell cat plugin.txt)
plugin_version :=  $(shell cat version.txt)
plugin_package := github.com/hdecarne/$(plugin_name)-telegraf-plugin/plugins/inputs/$(plugin_name)
plugin_cmd := $(plugin_name)-telegraf-plugin
plugin_conf := $(plugin_name).conf

MAKEFLAGS += --no-print-directory

GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
GOBIN ?= $(shell go env GOPATH)/bin

.DEFAULT_GOAL := build

deps:
	go mod download -x

testdeps: deps
	go install honnef.co/go/tools/cmd/staticcheck@2022.1.3

tidy:
	go mod verify
	go mod tidy

build: deps
ifneq (windows, $(GOOS))
	go build -o build/bin/$(plugin_cmd) ./cmd/$(plugin_cmd)
else
	go build -o build/bin/$(plugin_cmd).exe ./cmd/$(plugin_cmd)
endif
	cp $(plugin_conf) build/bin/

dist: build
	mkdir -p build/dist
	tar czvf build/dist/$(plugin_name)-$(GOOS)-$(GOARCH)-$(plugin_version).tar.gz -C build/bin .
ifneq (, $(shell command -v zip 2>/dev/null))
	zip -j build/dist/$(plugin_name)-$(GOOS)-$(GOARCH)-$(plugin_version).zip build/bin/*
else ifneq (, $(shell command -v 7z 2>/dev/null))
	7z a -bd build/dist/$(plugin_name)-$(GOOS)-$(GOARCH)-$(plugin_version).zip ./build/bin/*
endif

vet: testdeps
	go vet ./...

staticcheck: testdeps
	$(GOBIN)/staticcheck ./...

lint: vet staticcheck

test:
	go test -v -covermode=atomic -coverprofile=coverage.out ./...

check: test lint

clean:
	go clean ./...
	rm -rf build
	rm -f *.out
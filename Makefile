NAME=dashboard
VERSION=$(shell cat VERSION)
.DEFAULT_GOAL=build

gen-version:
	bash generate_version.sh

build: gen-version fetch-deps install-tools format
	bash generate_bindata.sh
	go build ./cmd/gomon

install-tools:
	go get github.com/go-bindata/go-bindata
	go install github.com/go-bindata/go-bindata

fetch-deps:
	go get -v ./...

format:
	go fmt .

install:
	install gomon /usr/local/bin/

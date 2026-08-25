.PHONY: build test

build:
	sh scripts/build.sh

test:
	go test ./...

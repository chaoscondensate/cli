.PHONY: build test release-check release-snapshot

build:
	sh scripts/build.sh

test:
	go test ./...

release-check:
	goreleaser check

release-snapshot:
	goreleaser release --snapshot --clean --skip=publish

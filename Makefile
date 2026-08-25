.PHONY: build test license-check release-check release-snapshot

build:
	sh scripts/build.sh

test:
	go test ./...

license-check:
	reuse lint

release-check:
	goreleaser check

release-snapshot:
	goreleaser release --snapshot --clean --skip=publish,chocolatey

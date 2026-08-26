.PHONY: build test dogfood license-check release-check release-snapshot

build:
	sh scripts/build.sh

test:
	go test ./...

dogfood: build
	bash scripts/dogfood.sh dist/forecast-ledger

license-check:
	reuse lint

release-check:
	goreleaser check

release-snapshot:
	goreleaser release --snapshot --clean --skip=publish,chocolatey

// Package schema embeds the exact supported Forecast Ledger contract and its
// conformance fixtures. Runtime code must never fetch a floating schema.
package schema

import (
	"embed"
	"io/fs"
)

const (
	Version               = "1.1.0"
	Commit                = "c04c72a178c15cd6cbbdd2e8a7b743d58872a94a"
	SchemaSHA256          = "c478f0f568c0c746c343a308d0fcb53815f4c8b91b4666f8f784913ad9132d15"
	ReleaseArchiveSHA256  = "edb2e307a7ce55984d17306556f0538f49a3a2a9fa66c9bfec973c90f0cb88dd"
	ForecastSealProtocol  = "forecast-seal/v1"
	ForecastTargetProfile = "forecast-envelope/v1"
)

//go:embed vendor/forecast-ledger/v1.1.0/forecast-ledger.schema.json
var contract []byte

//go:embed testdata/forecast-ledger/v1.1.0/*
var conformance embed.FS

// Contract returns an independent copy of the embedded schema bytes.
func Contract() []byte {
	result := make([]byte, len(contract))
	copy(result, contract)
	return result
}

// Conformance returns the embedded v1.1.0 fixture directory.
func Conformance() fs.FS {
	result, err := fs.Sub(conformance, "testdata/forecast-ledger/v1.1.0")
	if err != nil {
		panic(err)
	}
	return result
}

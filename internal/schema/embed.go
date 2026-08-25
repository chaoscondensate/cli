// Package schema embeds the exact supported Forecast Ledger contract and its
// conformance fixtures. Runtime code must never fetch a floating schema.
package schema

import (
	"embed"
	"io/fs"
)

const (
	Version               = "1.0.0"
	Commit                = "e409463d702888fefd253b32f21b9b2f864aabed"
	SchemaSHA256          = "e63bdd01f0241aa4d94d5ccc45e84bcea70a6a7fd46ab77cff4802b3f8b8fc65"
	ReleaseArchiveSHA256  = "a3d6afcf8a3cd9b9e9a650ebac684cbe2f155a81db309797d77694b5f4b9bbda"
	ForecastSealProtocol  = "forecast-seal/v1"
	ForecastTargetProfile = "forecast-envelope/v1"
)

//go:embed vendor/forecast-ledger/v1.0.0/forecast-ledger.schema.json
var contract []byte

//go:embed testdata/forecast-ledger/v1.0.0/*
var conformance embed.FS

// Contract returns an independent copy of the embedded schema bytes.
func Contract() []byte {
	result := make([]byte, len(contract))
	copy(result, contract)
	return result
}

// Conformance returns the embedded v1.0.0 fixture directory.
func Conformance() fs.FS {
	result, err := fs.Sub(conformance, "testdata/forecast-ledger/v1.0.0")
	if err != nil {
		panic(err)
	}
	return result
}

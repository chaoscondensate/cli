// Package schema embeds the exact supported Forecast Ledger contract and its
// conformance fixtures. Runtime code must never fetch a floating schema.
package schema

import (
	"embed"
	"io/fs"
)

const (
	Version               = "1.2.0"
	Commit                = "6c2fe3df99223945b8d1613a03f95796b3c7d1e2"
	SchemaSHA256          = "d609982f0fcea1ce076fdb32b44ef0eebe3265754eea7065de9d78a857dab5b8"
	ReleaseArchiveSHA256  = "5081c740cef4c0063a77a7e4aa51e142d355a30c09d41be9d4acfd8f7356ef8e"
	ForecastSealProtocol  = "forecast-seal/v1"
	ForecastTargetProfile = "forecast-envelope/v1"
)

//go:embed vendor/forecast-ledger/v1.2.0/forecast-ledger.schema.json
var contract []byte

//go:embed testdata/forecast-ledger/v1.2.0/*
var conformance embed.FS

// Contract returns an independent copy of the embedded schema bytes.
func Contract() []byte {
	result := make([]byte, len(contract))
	copy(result, contract)
	return result
}

// Conformance returns the embedded v1.2.0 fixture directory.
func Conformance() fs.FS {
	result, err := fs.Sub(conformance, "testdata/forecast-ledger/v1.2.0")
	if err != nil {
		panic(err)
	}
	return result
}

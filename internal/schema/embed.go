// Package schema embeds the exact supported Forecast Ledger contract and its
// conformance fixtures. Runtime code must never fetch a floating schema.
package schema

import (
	"embed"
	"io/fs"
)

const (
	Version                = "1.3.0"
	Commit                 = "32218f682b3a650f41153e98817473bf429973a7"
	AnnotatedTagObject     = "d3d1f06a7f27501b1419eaf78fc4a48e51de9ee3"
	SchemaSHA256           = "f673e4f3fc867a83d8c42a6992c6020ea28359a293580c8c742fe9dcdcd8d2c1"
	ReleaseArchiveSHA256   = "3b6b9f274a67d2714edaa308f9aad51b218dbf24ed95de1a1340292ad1df1f2a"
	ReleaseChecksumsSHA256 = "6042508976246ddc62974ad3054dca9885525024d4bb543572b75b23c60ac284"
	ForecastSealProtocol   = "forecast-seal/v1"
	ForecastTargetProfile  = "forecast-envelope/v1"
)

//go:embed vendor/forecast-ledger/v1.3.0/forecast-ledger.schema.json
var contract []byte

//go:embed vendor/forecast-ledger/v1.3.0/LICENSE
var license []byte

//go:embed testdata/forecast-ledger/v1.3.0/*
var conformance embed.FS

// Contract returns an independent copy of the embedded schema bytes.
func Contract() []byte {
	result := make([]byte, len(contract))
	copy(result, contract)
	return result
}

// License returns an independent copy of the exact upstream license bytes.
func License() []byte {
	result := make([]byte, len(license))
	copy(result, license)
	return result
}

// Conformance returns the embedded v1.3.0 fixture directory.
func Conformance() fs.FS {
	result, err := fs.Sub(conformance, "testdata/forecast-ledger/v1.3.0")
	if err != nil {
		panic(err)
	}
	return result
}

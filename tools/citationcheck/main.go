package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/chaoscondensate/forecast-ledger/internal/releasecheck"
)

func main() {
	path := flag.String("file", "CITATION.cff", "path to CITATION.cff")
	version := flag.String("version", "", "release version without a leading v")
	date := flag.String("date", "", "release date in YYYY-MM-DD form")
	flag.Parse()

	if flag.NArg() != 0 || strings.TrimSpace(*version) == "" || strings.TrimSpace(*date) == "" {
		fmt.Fprintln(os.Stderr, "usage: go run ./tools/citationcheck --version VERSION --date YYYY-MM-DD")
		os.Exit(2)
	}
	if err := releasecheck.CheckCitation(*path, strings.TrimPrefix(*version, "v"), *date); err != nil {
		fmt.Fprintf(os.Stderr, "citation check failed: %v\n", err)
		os.Exit(1)
	}
}

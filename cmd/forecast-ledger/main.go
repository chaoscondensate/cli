// Command forecast-ledger manages Forecast Ledger files and evidence.
package main

import (
	"context"
	"os"

	cliadapter "github.com/chaoscondensate/forecast-ledger/internal/adapters/cli"
)

func main() {
	os.Exit(cliadapter.Run(context.Background(), os.Args, os.Stdin, os.Stdout, os.Stderr))
}

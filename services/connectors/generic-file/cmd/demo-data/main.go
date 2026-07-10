// demo-data writes a reproducible demo payment export in the interchange CSV
// shape. Drop the output into the connector's drop directory to exercise the
// full ingestion → posting → anchoring pipeline:
//
//	go run ./cmd/demo-data -rows 250 -fiscal-year 2026 -out dropbox/demo.csv
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/opentreasury/opentreasury/services/connectors/generic-file/internal/demodata"
)

func main() {
	rows := flag.Int("rows", 100, "number of demo payments")
	fiscalYear := flag.Int("fiscal-year", 2026, "fiscal year the payments fall in")
	seed := flag.Int64("seed", 1, "random seed (same seed → same file)")
	institutions := flag.String("institutions", "minfin,health,educ", "comma-separated institution codes")
	out := flag.String("out", "", "output file (default stdout)")
	flag.Parse()

	records := demodata.Generate(demodata.Config{
		Seed:         *seed,
		Rows:         *rows,
		FiscalYear:   *fiscalYear,
		Institutions: strings.Split(*institutions, ","),
	})

	output := os.Stdout
	if *out != "" {
		file, err := os.Create(*out) // #nosec G304 -- operator-supplied output path
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		defer func() { _ = file.Close() }()
		output = file
	}

	writer := csv.NewWriter(output)
	if err := writer.WriteAll(records); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

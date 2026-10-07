package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Marist-Community-and-Belonging/report-gen-pdf/reportgen"
)

func main() {
	inPath := flag.String("in", "", "report JSON to render")
	dataDir := flag.String("data", "", "dashboard-primitives data/processed dir")
	year := flag.Int("year", 2023, "collection year, used with -data")
	outPath := flag.String("out", "out/report.pdf", "output PDF path")
	typstDir := flag.String("typst-dir", "", "optional editable Typst project directory")
	flag.Parse()

	var report []byte
	var err error

	switch {
	case *dataDir != "":
		report, err = reportgen.BuildFromProcessed(*dataDir, *year)
	case *inPath != "":
		report, err = os.ReadFile(*inPath)
	default:
		fmt.Fprintln(os.Stderr, "usage: reportgen (-in report.json | -data path/to/processed [-year 2023]) [-out out/report.pdf] [-typst-dir out/report-typst]")
		os.Exit(2)
	}
	if err != nil {
		fatal(err)
	}

	pdf, err := reportgen.Generate(report, reportgen.Options{OutPath: *outPath, TypstDir: *typstDir})
	if err != nil {
		fatal(err)
	}
	fmt.Printf("%s (%d bytes)\n", *outPath, len(pdf))
	if *typstDir != "" {
		fmt.Printf("%s (Typst project)\n", *typstDir)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

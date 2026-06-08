// Command libraw-api-inventory generates and checks LibRaw API inventory docs.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"

	"github.com/ivanglie/go-libraw/internal/apiinventory"
)

func main() {
	var headers string
	var coveragePath string
	var outPath string
	var check bool
	var updateCoverage bool

	flag.StringVar(&headers, "headers", "", "LibRaw header directory or install prefix")
	flag.StringVar(&coveragePath, "coverage", "internal/apiinventory/coverage.tsv", "coverage map TSV path")
	flag.StringVar(&outPath, "out", "docs/libraw-api-inventory.md", "inventory Markdown output path")
	flag.BoolVar(&check, "check", false, "verify output is current and every symbol has coverage")
	flag.BoolVar(&updateCoverage, "update-coverage", false, "rewrite coverage TSV, adding missing symbols as deferred")
	flag.Parse()

	if err := run(headers, coveragePath, outPath, check, updateCoverage); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(headers, coveragePath, outPath string, check, updateCoverage bool) error {
	inv, err := apiinventory.Generate(apiinventory.Options{HeaderDir: headers})
	if err != nil {
		return err
	}

	coverage := map[string]apiinventory.CoverageEntry{}
	if _, err := os.Stat(coveragePath); err == nil {
		coverage, err = apiinventory.LoadCoverage(coveragePath)
		if err != nil {
			return fmt.Errorf("load coverage map: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat coverage map: %w", err)
	}

	if updateCoverage {
		body := apiinventory.RenderCoverageTSV(inv, coverage)
		if err := os.WriteFile(coveragePath, body, 0o644); err != nil {
			return fmt.Errorf("write coverage map: %w", err)
		}
		coverage, err = apiinventory.LoadCoverage(coveragePath)
		if err != nil {
			return fmt.Errorf("reload coverage map: %w", err)
		}
	}

	body, missing := apiinventory.RenderMarkdown(inv, coverage)
	if len(missing) > 0 {
		for _, sym := range missing {
			fmt.Fprintf(os.Stderr, "missing coverage: %s %s from %s\n", sym.Kind, sym.Name, sym.Header)
		}
		return fmt.Errorf("%d upstream symbols missing coverage entries", len(missing))
	}

	if check {
		current, err := os.ReadFile(outPath)
		if err != nil {
			return fmt.Errorf("read inventory output: %w", err)
		}
		if !bytes.Equal(current, body) {
			return fmt.Errorf("%s is stale; run `make api-inventory`", outPath)
		}
		return nil
	}

	if err := os.WriteFile(outPath, body, 0o644); err != nil {
		return fmt.Errorf("write inventory output: %w", err)
	}
	return nil
}

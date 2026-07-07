// Command hotspot ranks the files in a repository by refactoring risk
// (git churn × code complexity) and prints them highest-risk first.
//
// Usage:
//
//	hotspot [flags] [path]
//
//	-top N          show the top N files (default 20; 0 = all)
//	-min-score F    hide files scoring below F (0..1)
//	-json           emit the full report as JSON
//	-include-untracked  also score files not tracked by git
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/richardwooding/hotspot"
)

func main() {
	top := flag.Int("top", 20, "show the top N files (0 = all)")
	minScore := flag.Float64("min-score", 0, "hide files scoring below this (0..1)")
	asJSON := flag.Bool("json", false, "emit the full report as JSON")
	untracked := flag.Bool("include-untracked", false, "also score files not tracked by git")
	flag.Parse()

	root := "."
	if flag.NArg() > 0 {
		root = flag.Arg(0)
	}

	rep, err := hotspot.Analyze(context.Background(), root, hotspot.Options{
		IncludeUntracked: *untracked,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "hotspot:", err)
		os.Exit(1)
	}

	filtered := rep.Files[:0]
	for _, f := range rep.Files {
		if f.Score >= *minScore {
			filtered = append(filtered, f)
		}
	}
	rep.Files = filtered
	if *top > 0 && len(rep.Files) > *top {
		rep.Files = rep.Files[:*top]
	}

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(rep); err != nil {
			fmt.Fprintln(os.Stderr, "hotspot:", err)
			os.Exit(1)
		}
		return
	}

	printTable(rep)
}

func printTable(rep hotspot.Report) {
	if len(rep.Files) == 0 {
		fmt.Println("No hotspots found (no analyzable source files, or all below --min-score).")
		return
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "SCORE\tCHURN\tCOGN\tCYC\tCA\tCE\tFNS\tLANG\tFILE")
	for _, f := range rep.Files {
		fmt.Fprintf(tw, "%.3f\t%d\t%d\t%d\t%d\t%d\t%d\t%s\t%s\n",
			f.Score, f.Commits, f.Cognitive, f.Cyclomatic, f.Afferent, f.Efferent, f.Functions, f.Language, f.Path)
	}
	_ = tw.Flush()
}

package effio

import (
	"fmt"
	"log"
)

func (cmd *Cmd) SummarizeCSV() {
	var samplesFlag, hbktFlag int
	var inFlag, outFlag string

	// TODO: add -json flag
	fs := cmd.FlagSet
	fs.StringVar(&inFlag, "in", "", "CSV file to load")
	fs.IntVar(&hbktFlag, "hbkt", 10, "number of histogram buckets")
	fs.StringVar(&outFlag, "out", "", "CSV file to write")
	fs.IntVar(&samplesFlag, "samples", 1, "Number of samples to write to the new file.")
	fs.Parse(cmd.Args)

	if samplesFlag < 1 {
		log.Fatalf("-samples must be >= 1")
	}

	recs := LoadCSV(inFlag)
	summary := recs.Summarize(samplesFlag, hbktFlag)
	printSummary(summary)
}

func printSummary(summary LatData) {
	// TODO: consider printing only integers?
	fmt.Printf("Min:                %g\n", summary.Min)
	fmt.Printf("Max:                %g\n", summary.Max)
	fmt.Printf("Count:              %d\n", summary.Count)
	fmt.Printf("Sum:                %f\n", summary.Sum)
	fmt.Printf("Average:            %g\n", summary.Average)
	fmt.Printf("Standard Deviation: %g\n", summary.Stddev)
	fmt.Printf("Variance:           %g\n", summary.Variance)
	fmt.Printf("Begin Timestamp:    %g\n", summary.BeginTs)
	fmt.Printf("End Timestamp:      %g\n", summary.EndTs)
	fmt.Printf("Elapsed Time:       %g\n", summary.ElapsedTime)
	fmt.Printf("P1:  %g P5:  %g P10: %g\n", summary.P1, summary.P5, summary.P10)
	fmt.Printf("P25: %g P50: %g P75: %g\n", summary.P25, summary.P50, summary.P75)
	fmt.Printf("P90: %g P95: %g P99: %g\n", summary.P90, summary.P95, summary.P99)

	fmt.Printf("Histogram: ")
	for _, lr := range summary.Histogram {
		fmt.Printf("%g ", lr.Val)
	}
	fmt.Printf("\nRead Histogram: ")
	for _, lr := range summary.RHistogram {
		fmt.Printf("%g ", lr.Val)
	}
	fmt.Printf("\nWrite Histogram: ")
	for _, lr := range summary.WHistogram {
		fmt.Printf("%g ", lr.Val)
	}
	fmt.Printf("\n")
	// leave trim out for now, none of my tests use it yet
}

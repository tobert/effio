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
	fmt.Printf("Samples:            %d\n", summary.Samples)
	fmt.Printf("Sum:                %f\n", summary.Sum)
	fmt.Printf("Average:            %g\n", summary.Average)
	fmt.Printf("Standard Deviation: %g\n", summary.Stddev)
	fmt.Printf("Variance:           %g\n", summary.Variance)
	fmt.Printf("Begin Timestamp:    %g\n", summary.BeginTs)
	fmt.Printf("End Timestamp:      %g\n", summary.EndTs)
	fmt.Printf("Elapsed Time:       %g\n", summary.ElapsedTime)
	fmt.Printf("\n")
	fmt.Printf("P1:  %8.2f P5:  %8.2f P10: %8.2f\n", summary.P1, summary.P5, summary.P10)
	fmt.Printf("P25: %8.2f P50: %8.2f P75: %8.2f\n", summary.P25, summary.P50, summary.P75)
	fmt.Printf("P90: %8.2f P95: %8.2f P99: %8.2f\n", summary.P90, summary.P95, summary.P99)

	fmt.Printf("\nHistogram:       ")
	for _, lr := range summary.Histogram {
		fmt.Printf("%8.2f ", lr.Val)
	}
	fmt.Printf("\nRead Histogram:  ")
	for _, lr := range summary.RHistogram {
		fmt.Printf("%8.2f ", lr.Val)
	}
	fmt.Printf("\nWrite Histogram: ")
	for _, lr := range summary.WHistogram {
		fmt.Printf("%8.2f ", lr.Val)
	}
	fmt.Printf("\n")
	// leave trim out for now, none of my tests use it yet
}

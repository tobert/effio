package effio

import (
	"encoding/json"
	"fmt"
	"os"
)

func (cmd *Cmd) SummarizeCSV() {
	var hbktFlag int
	var inFlag, outFlag string
	var jsonFlag bool

	// TODO: add -json flag
	cmd.DefaultFlags()
	cmd.FlagSet.StringVar(&inFlag, "in", "", "CSV file to load")
	cmd.FlagSet.IntVar(&hbktFlag, "hbkt", 10, "number of histogram buckets")
	cmd.FlagSet.StringVar(&outFlag, "out", "", "CSV file to write")
	cmd.FlagSet.BoolVar(&jsonFlag, "json", false, "Print JSON instead of human-readable text.")
	cmd.ParseArgs()

	recs := LoadFioLatlog(inFlag)
	smry := recs.Summarize(hbktFlag)

	if jsonFlag {
		printJson(smry)
	} else {
		printSummary(smry)
	}
}

func printJson(smry LatSummaries) {
	js, err := json.MarshalIndent(smry, "", "\t")
	if err != nil {
		fmt.Printf("Failed to encode summary data as JSON: %s\n", err)
		os.Exit(1)
	}
	js = append(js, byte('\n'))

	os.Stdout.Write(js)
}

func printSummary(smry LatSummaries) {
	fmt.Printf("Min:                %d\n", smry.Summary.Min)
	fmt.Printf("Max:                %d\n", smry.Summary.Max)
	fmt.Printf("Count:              %d\n", smry.Summary.Count)
	fmt.Printf("Sum:                %d\n", smry.Summary.Sum)
	fmt.Printf("Average:            %g\n", smry.Summary.Average)
	fmt.Printf("Standard Deviation: %g\n", smry.Summary.Stdev)
	fmt.Printf("Begin Timestamp:    %d\n", smry.Summary.MinTs)
	fmt.Printf("End Timestamp:      %d\n", smry.Summary.MaxTs)
	fmt.Printf("Elapsed Time:       %d\n", smry.Summary.Elapsed)
	fmt.Printf("\n")
	fmt.Printf("P1:    % 8d P5:     % 8d P10:     % 8d\n", smry.Pcntl[1].Val, smry.Pcntl[5].Val, smry.Pcntl[10].Val)
	fmt.Printf("P25:   % 8d P50:    % 8d P75:     % 8d\n", smry.Pcntl[25].Val, smry.Pcntl[50].Val, smry.Pcntl[75].Val)
	fmt.Printf("P90:   % 8d P95:    % 8d P99:     % 8d\n", smry.Pcntl[90].Val, smry.Pcntl[95].Val, smry.Pcntl[99].Val)
	fmt.Printf("P99.9: % 8d P99.99: % 8d P99.999: % 8d\n", smry.Pcntl[99.9].Val, smry.Pcntl[99.99].Val, smry.Pcntl[99.999].Val)

	fmt.Printf("\nAll Histogram[% 4d]:   ", len(smry.Histogram))
	for _, bkt := range smry.Histogram {
		fmt.Printf("% 10d ", bkt.Average)
	}
	fmt.Printf("\nRead Histogram[% 4d]:  ", len(smry.RHistogram))
	for _, bkt := range smry.RHistogram {
		fmt.Printf("% 10d ", bkt.Average)
	}
	fmt.Printf("\nWrite Histogram[% 4d]: ", len(smry.WHistogram))
	for _, bkt := range smry.WHistogram {
		fmt.Printf("% 10d ", bkt.Average)
	}
	fmt.Printf("\n")
	// leave trim out for now, none of my tests use it yet
}

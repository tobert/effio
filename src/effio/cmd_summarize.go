package effio

import (
	"encoding/json"
	"log"
	"os"
)

func (cmd *Cmd) SummarizeCSV() {
	var samplesFlag, hbktFlag int
	var inFlag, outFlag string

	fs := cmd.FlagSet
	fs.StringVar(&inFlag, "in", "", "CSV file to load")
	fs.IntVar(&hbktFlag, "hbkt", 10, "number of histogram buckets")
	fs.StringVar(&outFlag, "out", "", "CSV file to write")
	fs.IntVar(&samplesFlag, "samples", 0, "Number of samples to write to the new file.")
	fs.Parse(cmd.Args)

	recs := LoadCSV(inFlag)
	summary := recs.Summarize(samplesFlag, hbktFlag)
	js, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		log.Fatalf("Could not serialize summary to JSON: %s\n", err)
	}
	os.Stdout.Write(js)
	os.Stdout.WriteString("\n")
}

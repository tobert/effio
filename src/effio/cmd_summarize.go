package effio

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
)

func (cmd *Cmd) SummarizeCSV() {
	var hbktFlag int
	var inFlag, outFlag string
	var jsonFlag bool

	cmd.DefaultFlags()
	cmd.FlagSet.StringVar(&inFlag, "in", "", "CSV file to load")
	cmd.FlagSet.IntVar(&hbktFlag, "hbkt", 10, "number of histogram buckets")
	cmd.FlagSet.StringVar(&outFlag, "out", "", "CSV file to write")
	cmd.FlagSet.BoolVar(&jsonFlag, "json", false, "Print JSON instead of human-readable text.")
	cmd.ParseArgs()

	recs := LoadFioLatlog(inFlag)
	smry := recs.Summarize(hbktFlag)
	AppendMetadata(inFlag, &smry)

	if jsonFlag {
		os.Stdout.Write(toJson(smry))
	} else {
		printSummary(smry)
	}
}

// effio summarize-all -path suites -out public/data
func (cmd *Cmd) SummarizeAll() {
	var hbktFlag int
	var outFlag string

	cmd.DefaultFlags()
	cmd.FlagSet.IntVar(&hbktFlag, "hbkt", 10, "number of histogram buckets")
	cmd.FlagSet.StringVar(&outFlag, "out", "public/data", "directory to write summaries to")
	cmd.ParseArgs()

	fi, err := os.Stat(outFlag)
	if err != nil {
		log.Fatalf("Could not stat '%s': %s\n", outFlag, err)
	}
	if !fi.IsDir() {
		log.Fatalf("'%s' must be a directory!\n", outFlag)
	}

	files := InventoryCSVFiles(cmd.PathFlag)

	for _, file := range files {
		recs := LoadFioLatlog(file)
		smry := recs.Summarize(hbktFlag)
		AppendMetadata(file, &smry)

		// output filename is SHA1 of the source file
		sha1sum := sha1file(file)
		outpath := path.Join(outFlag, fmt.Sprintf("%s.json", sha1sum))

		out, err := os.OpenFile(outpath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			log.Fatalf("Could not open file '%s' for write: %s\n", outpath, err)
		}
		out.Write(toJson(smry))
	}
}

func InventoryCSVFiles(dpath string) []string {
	out := make([]string, 0)
	wanted := []string{"bw_bw.log", "lat_lat.log", "lat_slat.log", "lat_clat.log", "iops_iops.log"}

	visitor := func(dpath string, f os.FileInfo, err error) error {
		if err != nil {
			log.Fatalf("Encountered an error while inventorying files in '%s': %s", dpath, err)
		}

		fi, err := os.Stat(dpath)
		if err != nil {
			log.Fatalf("Could not stat '%s': %s\n", dpath, err)
		}

		// skip empty and tiny files
		if fi.Size() < 100 {
			return nil
		}

		for _, want := range wanted {
			if path.Base(dpath) == want {
				out = append(out, dpath)
			}
		}

		return nil
	}

	err := filepath.Walk(dpath, visitor)
	if err != nil {
		log.Fatalf("Could not inventory files in '%s': %s", dpath, err)
	}

	return out
}

func toJson(smry LatSummaries) []byte {
	js, err := json.MarshalIndent(smry, "", "\t")
	if err != nil {
		fmt.Printf("Failed to encode summary data as JSON: %s\n", err)
		os.Exit(1)
	}
	return append(js, byte('\n'))
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
		fmt.Printf("% 7.3f ", bkt.Average)
	}
	fmt.Printf("\nRead Histogram[% 4d]:  ", len(smry.RHistogram))
	for _, bkt := range smry.RHistogram {
		fmt.Printf("% 7.3f ", bkt.Average)
	}
	fmt.Printf("\nWrite Histogram[% 4d]: ", len(smry.WHistogram))
	for _, bkt := range smry.WHistogram {
		fmt.Printf("% 7.3f ", bkt.Average)
	}
	fmt.Printf("\nTrim Histogram[% 4d]:  ", len(smry.THistogram))
	for _, bkt := range smry.THistogram {
		fmt.Printf("% 7.3f ", bkt.Average)
	}
	fmt.Printf("\n")
}

func AppendMetadata(dpath string, smry *LatSummaries) {
	fcmd_filenames := []string{"command.json", "test.json"}
	dir := path.Dir(dpath)

	for _, name := range fcmd_filenames {
		fpath := path.Join(dir, name)
		if fi, err := os.Stat(fpath); err == nil {
			if fi.Size() > 0 {
				smry.FioCommand = LoadFioCommandJson(fpath)
			}
		}
	}

	fpath := path.Join(dir, "output.json")
	if fi, err := os.Stat(fpath); err == nil {
		if fi.Size() > 0 {
			smry.FioJsonData = LoadFioJsonData(fpath)
		}
	}
}

func sha1file(file string) string {
	hasher := sha1.New()

	f, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	if _, err := io.Copy(hasher, f); err != nil {
		log.Fatal(err)
	}

	return fmt.Sprintf("%x", hasher.Sum(nil))
}

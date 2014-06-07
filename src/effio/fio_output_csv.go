package effio

// encoding/csv doesn't strip whitespace and does a fair bit of
// work to handle strings & quoting which are totally unnecessary
// overhead for these files so skip it

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// the input is ints but plotinum uses float64 so might as well
// start there and avoid the type conversions later
type LatRec struct {
	time float64 // time offset from beginning of fio run
	perf float64 // latency value
	ddir uint8   // 0 = read, 1 = write, 2 = trim
	bsz  uint16  // block size
}

type LatRecs []LatRec

// Loads the CSV output by fio into an LatRecs array of LatRec structs.
func LoadCSV(filename string) LatRecs {
	fmt.Printf("Parsing file: '%s' ... ", filename)

	fd, err := os.Open(filename)
	if err != nil {
		fmt.Printf(" Failed.\nCould not open file '%s' for read: %s\n", filename, err)
		return LatRecs{}
	}
	defer fd.Close()

	started := time.Now()
	records := make(LatRecs, 0)
	var tm, perf float64
	var ddir, bsz int
	bfd := bufio.NewReader(fd)
	var lno int = 0
	for {
		line, _, err := bfd.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("\nRead from file '%s' failed: %s", filename, err)
		}
		lno++

		if lno%10000 == 0 {
			fmt.Printf(".")
		}

		// fio always uses ", " instead of "," as far as I can tell
		r := strings.SplitN(string(line), ", ", 4)
		// probably an impartial record at the end of the file
		if len(r) < 4 || r[0] == "" || r[1] == "" {
			continue
		}

		tm, err = strconv.ParseFloat(r[0], 64)
		if err != nil {
			log.Fatalf("\nParsing field 0 failed in file '%s' at line %d: %s", filename, lno, err)
		}
		perf, err = strconv.ParseFloat(r[1], 64)
		if err != nil {
			log.Fatalf("\nParsing field 1 in file '%s' at line %d: %s", filename, lno, err)
		}
		ddir, err = strconv.Atoi(r[2])
		if err != nil {
			log.Fatalf("\nParsing field 2 failed in file '%s' at line %d: %s", filename, lno, err)
		}
		bsz, err = strconv.Atoi(r[3])
		if err != nil {
			log.Fatalf("\nParsing field 3 failed in file '%s' at line %d: %s", filename, lno, err)
		}

		lr := LatRec{tm, perf, uint8(ddir), uint16(bsz)}
		records = append(records, lr)
	}

	done := time.Now()
	fmt.Printf(" Done.\nRows: %d Elapsed: %s\n", len(records), done.Sub(started).String())

	return records
}

// implement some plotinum interfaces
func (lrs LatRecs) Len() int {
	return len(lrs)
}

func (lrs LatRecs) XY(i int) (float64, float64) {
	return lrs[i].time, lrs[i].perf
}

func (lrs LatRecs) Value(i int) float64 {
	return lrs[i].perf
}

func (lrs LatRecs) Values(i int) (vals []float64) {
	for _, l := range lrs {
		vals = append(vals, l.perf)
	}
	return
}

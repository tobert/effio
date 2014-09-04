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

// Loads the CSV output by fio into an LogRecs array of LogRec structs.
func LoadFioLog(filename string) LogRecs {
	fmt.Printf("Parsing file: '%s' ... ", filename)

	fd, err := os.Open(filename)
	if err != nil {
		fmt.Printf(" Failed.\nCould not open file '%s' for read: %s\n", filename, err)
		return LogRecs{}
	}
	defer fd.Close()

	started := time.Now()
	records := make(LogRecs, 0)

	bfd := bufio.NewReader(fd)
	lno := 0
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

		tm, err := strconv.Atoi(r[0])
		if err != nil {
			log.Fatalf("\nParsing field 0 failed in file '%s' at line %d: %s", filename, lno, err)
		}
		perf, err := strconv.Atoi(r[1])
		if err != nil {
			log.Fatalf("\nParsing field 1 in file '%s' at line %d: %s", filename, lno, err)
		}
		ddir, err := strconv.Atoi(r[2])
		if err != nil {
			log.Fatalf("\nParsing field 2 failed in file '%s' at line %d: %s", filename, lno, err)
		}
		bsz, err := strconv.Atoi(r[3])
		if err != nil {
			log.Fatalf("\nParsing field 3 failed in file '%s' at line %d: %s", filename, lno, err)
		}

		lr := LogRec{uint32(tm), uint32(perf), uint8(ddir), uint16(bsz), uint32(lno)}
		records = append(records, &lr)
	}

	done := time.Now()
	fmt.Printf(" Done.\nRows: %d Elapsed: %s\n", len(records), done.Sub(started).String())

	return records
}

func (lrs LogRecs) DumpCSV(fpath string) {
	fd, err := os.OpenFile(fpath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalf("Could not open '%s' for write: %s\n", fpath, err)
	}
	defer fd.Close()

	for _, lr := range lrs {
		// TODO: Something isn't right with the sampling below
		// all the samples should always be full
		if lr == nil {
			break
		}
		fmt.Fprintf(fd, "%f,%f,%d,%d\n", lr.Time, lr.Val, lr.Ddir, lr.Bsz)
	}
}

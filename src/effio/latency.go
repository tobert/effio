package effio

// encoding/csv doesn't strip whitespace and does a fair bit of
// work to handle strings & quoting which are totally unnecessary
// overhead for these files so skip it

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

// the input is ints but plotinum uses float64 so might as well
// start there and avoid the type conversions later
type LatRec struct {
	Time float64 `json:"x"` // time offset from beginning of fio run
	Val  float64 `json:"y"` // latency value
	Ddir uint8   `json:"-"` // 0 = read, 1 = write, 2 = trim
	Bsz  uint16  `json:"-"` // block size
}
type LatRecs []*LatRec

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
		records = append(records, &lr)
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
	return lrs[i].Time, lrs[i].Val
}

func (lrs LatRecs) Value(i int) float64 {
	return lrs[i].Val
}

func (lrs LatRecs) Values(i int) (vals []float64) {
	for _, l := range lrs {
		vals = append(vals, l.Val)
	}
	return
}

func (lrs LatRecs) DumpCSV(fpath string) {
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
		fmt.Fprintf(fd, "%f,%f,%d\n", lr.Time, lr.Val, lr.Ddir)
	}
}

type LatData struct {
	Min         float64 `json:"min"`
	Max         float64 `json:"max"`
	Count       int     `json:"count"`
	Sum         float64 `json:"sum"`
	Average     float64 `json:"average"`
	Stddev      float64 `json:"stddev"`
	Variance    float64 `json:"variance"`
	P1          float64 `json:"p1"`
	P5          float64 `json:"p5"`
	P10         float64 `json:"p10"`
	P25         float64 `json:"p25"`
	P50         float64 `json:"p50"`
	P75         float64 `json:"p75"`
	P90         float64 `json:"p90"`
	P95         float64 `json:"p95"`
	P99         float64 `json:"p99"`
	BeginTs     float64 `json:"first_ts"`
	EndTs       float64 `json:"last_ts"`
	ElapsedTime float64 `json:"elapsed"`
	Histogram   LatRecs `json:"histogram"`
	RHistogram  LatRecs `json:"read_histogram"`
	WHistogram  LatRecs `json:"write_histogram"`
	THistogram  LatRecs `json:"trim_histogram"`
	RecSm       LatRecs `json:"-"` // summarized to summary_size records (mean)
	RRecSm      LatRecs `json:"-"` // summarized to summary_size records (mean)
	WRecSm      LatRecs `json:"-"` // summarized to summary_size records (mean)
	TRecSm      LatRecs `json:"-"` // summarized to summary_size records (mean)
}

// Summarizes the LatRecs data into a LatData.
// First argument is the number of samples to put in the summaries.
// Second argument is the number of buckets in the histograms.
// This does all the work in 3 passes, the first getting avg/min/max.
// Then the values are sorted to access the percentiles by index.
// The final pass computes the standard deviation, which requires the average
// from the first pass.
func (lrs LatRecs) Summarize(summary_size int, histogram_size int) (ld LatData) {
	ld.Max = math.SmallestNonzeroFloat64
	ld.Min = math.MaxFloat64
	ld.BeginTs = lrs[0].Time
	ld.EndTs = lrs[len(lrs)-1].Time
	ld.ElapsedTime = math.Abs(ld.BeginTs - ld.EndTs)

	// allocate histograms & summaries
	// same thing really but summaries are meant to be used to take huge
	// CSV logs and reduce them down to something more manageable
	ld.RecSm = make(LatRecs, summary_size)       // all-IO sampled data
	ld.RRecSm = make(LatRecs, summary_size)      // reads sampled data
	ld.WRecSm = make(LatRecs, summary_size)      // writes sampled data
	ld.TRecSm = make(LatRecs, summary_size)      // trims sampled data
	ld.Histogram = make(LatRecs, histogram_size) // all-IO histogram
	ld.RHistogram = make(LatRecs, histogram_size) // reads histogram
	ld.WHistogram = make(LatRecs, histogram_size) // writes histogram
	ld.THistogram = make(LatRecs, histogram_size) // trims histogram

	if summary_size > len(lrs) {
		summary_size = len(lrs)
	}

	if histogram_size > len(lrs) {
		histogram_size = len(lrs)
	}

	// variables needed for creating all-IO summaries & histograms
	var reads, writes, trims int                                  // count up by IO direction
	var arec, acnt, ahgrec, ahgcnt int                            // indexes into buckets/output
	abkt := make(LatRecs, bucketSize(summary_size, len(lrs)))     // summary bucket
	ahgbkt := make(LatRecs, bucketSize(histogram_size, len(lrs))) // histogram bucket

	// list of values, to be sorted for getting percentiles
	lvs := make([]float64, len(lrs))

	// first pass
	for i, lr := range lrs {
		ld.Count++
		ld.Sum += lr.Val

		if lr.Val > ld.Max {
			ld.Max = lr.Val
		}

		if lr.Val < ld.Min {
			ld.Min = lr.Val
		}

		// create all-IO sample/summary & histogram
		arec, acnt = abkt.updateBucket(arec, acnt, ld.RecSm, lr)
		ahgrec, ahgcnt = ahgbkt.updateBucket(ahgrec, ahgcnt, ld.Histogram, lr)

		// count up each by IO type for resampling/histograms
		if lr.Ddir == 0 {
			reads++
		} else if lr.Ddir == 1 {
			writes++
		} else if lr.Ddir == 2 {
			trims++
		}

		lvs[i] = lr.Val // for sorting on value for percentiles
	}

	// needed for stddev
	ld.Average = ld.Sum / float64(ld.Count)

	// buckets / indexes / counts for summarization
	var rrec, wrec, trec, rcnt, wcnt, tcnt int
	rbkt := make(LatRecs, bucketSize(summary_size, reads))
	wbkt := make(LatRecs, bucketSize(summary_size, writes))
	tbkt := make(LatRecs, bucketSize(summary_size, trims))

	// used to build histograms, same as summarization, but (usually) much smaller
	var rhgrec, rhgcnt, whgrec, whgcnt, thgrec, thgcnt int
	rhgbkt := make(LatRecs, bucketSize(histogram_size, reads))
	whgbkt := make(LatRecs, bucketSize(histogram_size, writes))
	thgbkt := make(LatRecs, bucketSize(histogram_size, trims))

	var dsum float64 // sum for stddev

	// second pass, populate ddir summaries/histograms & build stddev sum
	for _, lr := range lrs {
		if lr.Ddir == 0 {
			rrec, rcnt = rbkt.updateBucket(rrec, rcnt, ld.RRecSm, lr)
			rhgrec, rhgcnt = rhgbkt.updateBucket(rhgrec, rhgcnt, ld.RHistogram, lr)
		} else if lr.Ddir == 1 {
			wrec, wcnt = wbkt.updateBucket(wrec, wcnt, ld.WRecSm, lr)
			whgrec, whgcnt = whgbkt.updateBucket(whgrec, whgcnt, ld.WHistogram, lr)
		} else if lr.Ddir == 2 {
			trec, tcnt = tbkt.updateBucket(trec, tcnt, ld.TRecSm, lr)
			thgrec, thgcnt = thgbkt.updateBucket(thgrec, thgcnt, ld.THistogram, lr)
		}

		// update stddev sum
		dsum += math.Pow((lr.Val - ld.Average), 2)
	}

	// finish computing variance & standard deviation
	ld.Variance = dsum / float64(ld.Count)
	ld.Stddev = math.Sqrt(ld.Variance)

	// sort []float64 list then assign percentiles
	sort.Float64s(lvs)
	pctl_idx := func(pc float64) int {
		idx := math.Floor(float64(len(lvs))*(pc/100) + 0.5)
		out := int(idx)
		return out
	}

	ld.P1 = lvs[pctl_idx(1)]
	ld.P5 = lvs[pctl_idx(5)]
	ld.P10 = lvs[pctl_idx(10)]
	ld.P25 = lvs[pctl_idx(25)]
	ld.P50 = lvs[pctl_idx(50)]
	ld.P75 = lvs[pctl_idx(75)]
	ld.P90 = lvs[pctl_idx(90)]
	ld.P95 = lvs[pctl_idx(95)]
	ld.P99 = lvs[pctl_idx(99)]

	return
}

// write the latdata + CSV summaries to the specified path + filename fragment
// e.g. summary-%s.json, summary-read-%s.csv, summary-write-%s.csv, summary-trim-%s.csv
func (ld *LatData) WriteFiles(fpath string, ffrag string) {
	jsonpath := path.Join(fpath, fmt.Sprintf("summary-%s.json", ffrag))
	fd, err := os.OpenFile(jsonpath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalf("Could not open '%s' for write: %s\n", jsonpath, err)
	}
	defer fd.Close()

	enc := json.NewEncoder(fd)

	err = enc.Encode(ld)
	if err != nil {
		log.Fatalf("JSON encoding failed on file '%s': %s\n", jsonpath, err)
	}

	ld.RRecSm.DumpCSV(path.Join(fpath, fmt.Sprintf("summary-read-%s.csv", ffrag)))
	ld.WRecSm.DumpCSV(path.Join(fpath, fmt.Sprintf("summary-write-%s.csv", ffrag)))
	ld.TRecSm.DumpCSV(path.Join(fpath, fmt.Sprintf("summary-trim-%s.csv", ffrag)))
}

// compute the bucket size, default to 1 if less than summary_size
func bucketSize(buckets int, available int) int {
	if buckets < available {
		return int(math.Floor(float64(available) / float64(buckets)))
	}
	return 1
}

// Adds the value to the bucket at index bktidx, with lr. When full
// summarized into smry[smry_idx]. Returns updated indexes.
// it is safe to use the same bucket on each iteration
// bktidx: current bucket index
// hgidx: current histogram index
// hgram: histogram (list) - written to!
// lr: current latency record to add to the bucket
// Returns: (new bucket index, new histogram index)
func (bucket LatRecs) updateBucket(bktidx int, hgidx int, hgram LatRecs, lr *LatRec) (int, int) {
	// [..., bktidx => lr, ... ]
	bucket[bktidx] = lr
	bktidx++

	// advance the bucket index, stay on the same summary index
	if bktidx < len(bucket) {
		return bktidx, hgidx
	// bucket is full, sum it & advance to the next histogram entry
	} else {
		var ptotal, ttotal float64
		for _, v := range bucket {
			ptotal += v.Val
			ttotal += v.Time
		}

		nlr := LatRec{
			Time: math.Floor(ttotal / float64(len(bucket))),
			Val:  ptotal / float64(len(bucket)),
			Ddir: lr.Ddir,
			Bsz:  lr.Bsz,
		}

		hgram[hgidx] = &nlr

		// bucket is now summed & stored at hgidx, reset the bucket index to 0
		return 0, hgidx + 1
	}
}

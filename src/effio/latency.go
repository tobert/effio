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

// number of values to keep in summaries
const histogram_size = 10
const summary_size = 1000

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
// This does all the work in 3 passes, the first getting avg/min/max.
// Then the values are sorted to access the percentiles by index.
// The final pass computes the standard deviation, which requires the average
// from the first pass.
func (lrs LatRecs) Summarize() (ld LatData) {
	ld.Max = math.SmallestNonzeroFloat64
	ld.Min = math.MaxFloat64
	ld.BeginTs = lrs[0].Time
	ld.EndTs = lrs[len(lrs)-1].Time
	ld.ElapsedTime = math.Abs(ld.BeginTs - ld.EndTs)

	// compute the bucket size, default to 1 if less than 10k samples
	bucket_sz := 1
	if len(lrs) > summary_size {
		bucket_sz = int(math.Ceil(float64(len(lrs)) / summary_size))
	}
	hgram_bucket_sz := int(math.Ceil(float64(len(lrs)) / histogram_size))

	// buckets / indexes / counts for summarization
	var arec, rrec, wrec, trec, acnt, rcnt, wcnt, tcnt int
	abkt := make(LatRecs, bucket_sz)
	rbkt := make(LatRecs, bucket_sz)
	wbkt := make(LatRecs, bucket_sz)
	tbkt := make(LatRecs, bucket_sz)

	// summarized samples, a slice is saved to ld after the first pass
	arecsm := make(LatRecs, summary_size)
	// by ddir
	rrecsm := make(LatRecs, summary_size)
	wrecsm := make(LatRecs, summary_size)
	trecsm := make(LatRecs, summary_size)

	// used to build histograms
	var ahgrec, ahgcnt, rhgrec, rhgcnt, whgrec, whgcnt, thgrec, thgcnt int
	ahgbkt := make(LatRecs, hgram_bucket_sz)
	rhgbkt := make(LatRecs, hgram_bucket_sz)
	whgbkt := make(LatRecs, hgram_bucket_sz)
	thgbkt := make(LatRecs, hgram_bucket_sz)
	ld.Histogram = make(LatRecs, histogram_size)
	ld.RHistogram = make(LatRecs, histogram_size)
	ld.WHistogram = make(LatRecs, histogram_size)
	ld.THistogram = make(LatRecs, histogram_size)

	// list of values, to be sorted for getting percentiles
	lvs := make([]float64, len(lrs))

	for i, lr := range lrs {
		ld.Count++
		ld.Sum += lr.Val

		if lr.Val > ld.Max {
			ld.Max = lr.Val
		}

		if lr.Val < ld.Min {
			ld.Min = lr.Val
		}

		arec, acnt = abkt.updateBucket(arec, acnt, arecsm, lr)
		ahgrec, ahgcnt = ahgbkt.updateBucket(ahgrec, ahgcnt, ld.Histogram, lr)
		if lr.Ddir == 0 {
			rrec, rcnt = rbkt.updateBucket(rrec, rcnt, rrecsm, lr)
			rhgrec, rhgcnt = rhgbkt.updateBucket(rhgrec, rhgcnt, ld.RHistogram, lr)
		} else if lr.Ddir == 1 {
			wrec, wcnt = wbkt.updateBucket(wrec, wcnt, wrecsm, lr)
			whgrec, whgcnt = whgbkt.updateBucket(whgrec, whgcnt, ld.WHistogram, lr)
		} else if lr.Ddir == 2 {
			trec, tcnt = tbkt.updateBucket(trec, tcnt, trecsm, lr)
			thgrec, thgcnt = thgbkt.updateBucket(thgrec, thgcnt, ld.THistogram, lr)
		}

		lvs[i] = lr.Val // for sorting on value for percentiles
	}

	// there might be less than summary_size samples, save only the populated slice
	ld.RecSm = arecsm[0:acnt]
	ld.RRecSm = rrecsm[0:rcnt]
	ld.WRecSm = wrecsm[0:wcnt]
	ld.TRecSm = trecsm[0:tcnt]

	// sort then assign percentiles
	sort.Float64s(lvs)
	pctl_idx := func(pc float64) int {
		idx := math.Floor(float64(len(lvs))*(pc/100) + 0.5)
		out := int(idx)
		//fmt.Printf("pctl_idx(%f) = %f -> lvs[%d]\n", pc, idx, out)
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

	// needed for stddev
	ld.Average = ld.Sum / float64(ld.Count)

	// second pass over values is required to compute the standard deviation
	// use lvs instead of lrs since it was just sorted and might be in cache
	var dsum float64
	for _, v := range lvs {
		dsum += math.Pow((v - ld.Average), 2)
	}
	ld.Variance = dsum / float64(ld.Count)
	ld.Stddev = math.Sqrt(ld.Variance)

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

// Adds the value to the bucket at index bktidx, with lr. When full
// summarized into smry[smry_idx]. Returns updated indexes.
func (bucket LatRecs) updateBucket(bktidx int, smry_idx int, smry LatRecs, lr *LatRec) (int, int) {
	//fmt.Printf("(%d) bucket[%d] = %v\n", len(bucket), bktidx, lr)
	bucket[bktidx] = lr

	// end of the bucket
	if bktidx+1 == len(bucket) {
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
		//fmt.Printf("smry[%d] = %v (%d)\n", smry_idx, nlr, len(smry))
		smry[smry_idx] = &nlr

		// bucket is now summed & stored, reset the bucket index to 0
		return 0, smry_idx + 1
		// advance the bucket index, stay on the same summary index
	} else {
		return bktidx + 1, smry_idx
	}
}

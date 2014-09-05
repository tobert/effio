package effio

import (
	"fmt"
	"math"
	"sort"
)

// Log Record: The 4 fields from fio's latency logs and an index cache
// This is where most of the memory goes
type LogRec struct {
	Time uint32 `json:"time"`  // time offset from beginning of fio run
	Val  uint32 `json:"value"` // latency value in usec
	Ddir uint8  `json:"-"`     // 0 = read, 1 = write, 2 = trim
	Bsz  uint16 `json:"-"`     // block size
	Idx  uint32 `json:"-"`     // save the original index in LogRecs
}
type LogPcntl map[float64]*LogRec // .MarshalJSON() at EOF
type LogRecs []*LogRec

// sort interface impl, sorts by value for indexing percentiles
func (p LogRecs) Len() int           { return len(p) }
func (p LogRecs) Less(i, j int) bool { return p[i].Val < p[j].Val }
func (p LogRecs) Swap(i, j int)      { p[i].Val, p[j].Val = p[j].Val, p[i].Val }

// Log Bucket Summary: a handful of useful values for each bucket in
// the LogBin.
type LogSmry struct {
	Min     uint32   `json:"min"`
	Max     uint32   `json:"max"`
	Sum     uint64   `json:"sum"`
	Count   uint64   `json:"count"`
	Median  uint64   `json:"median"`
	Stdev   float64  `json:"stdev"`
	Average float64  `json:"average"`
	MinTs   uint32   `json:"min_ts"`
	MaxTs   uint32   `json:"max_ts"`
	Elapsed uint32   `json:"elapsed"`
	Pcntl   LogPcntl `json:"percentiles"`
}
type LogBin []*LogSmry

func NewLogBin(size int) LogBin {
	lhg := make(LogBin, size)
	for i, _ := range lhg {
		lhg[i] = &LogSmry{}
	}
	return lhg
}

type LogSummaries struct {
	Name    string `json:"name"`     // base name of the logfile (e.g. lat_lat.log)
	Path    string `json:"path"`     // full path to the file read
	LogType string `json:"log_type"` // e.g. bw, lat, slat, clat, iops
	// the fio command used to generate the file
	FioCommand FioCommand `json:"fio_command"`
	// data from the output of fio --output=json
	FioJsonData FioJsonData `json:"fio_data"`
	// the global summary
	Summary LogSmry `json:"summary"`
	// all 99 percentiles + 99.9, 99.99, and 99.999%
	Pcntl LogPcntl `json:"percentiles"`
	// bin across all samples, then by io direction
	Bin  LogBin `json:"bin"`       // binned version of all records
	RBin LogBin `json:"read_bin"`  // read ops
	WBin LogBin `json:"write_bin"` // write ops
	TBin LogBin `json:"trim_bin"`  // trim ops
	// capture outliers by preserving full resolution for metrics <P1 and >P99
	P1Bin   LogBin `json:"p1_bin"`        // bin from records with values < P1
	P1RBin  LogBin `json:"p1_read_bin"`   // <P1 / read
	P1WBin  LogBin `json:"p1_write_bin"`  // <P1 / write
	P1TBin  LogBin `json:"p1_trim_bin"`   // <P1 / trim
	P99Bin  LogBin `json:"p99_bin"`       // records with values > P99
	P99RBin LogBin `json:"p99_read_bin"`  // >P99 / read
	P99WBin LogBin `json:"p99_write_bin"` // >P99 / write
	P99TBin LogBin `json:"p99_trim_bin"`  // >P99 / trim
}

// Summarizes the LogRecs data into a LogSmry.
// First argument is the number of samples to put in the summaries.
// Second argument is the number of buckets in the bins.
// This does all the work in 3 passes, the first getting avg/min/max.
// Then the values are sorted to access the percentiles by index.
// The final pass computes the standard deviation, which requires the average
// from the first pass.
func (lrs LogRecs) Summarize(bins int) (ld LogSummaries) {
	if bins > len(lrs) {
		bins = len(lrs)
	}

	smry := LogSmry{
		Max:     0,
		Min:     math.MaxUint32,
		MinTs:   lrs[0].Time,
		MaxTs:   lrs[len(lrs)-1].Time,
		Elapsed: lrs[len(lrs)-1].Time - lrs[0].Time,
	}

	// count, sum, min, max
	for _, lr := range lrs {
		smry.Count++
		smry.Sum += uint64(lr.Val)

		if lr.Val > smry.Max {
			smry.Max = lr.Val
		}

		if lr.Val < smry.Min {
			smry.Min = lr.Val
		}
	}

	// average is required to compute stdev
	smry.Average = float64(smry.Sum) / float64(smry.Count)

	// second pass for stdev
	var dsum float64
	for _, lr := range lrs {
		dsum += math.Pow(float64(lr.Val)-smry.Average, 2)
	}

	// finish computing variance & standard deviation
	variance := dsum / float64(smry.Count)
	smry.Stdev = math.Sqrt(variance)

	// assign the completed summary to the return struct
	ld.Summary = smry

	// warning: will do some sorting on slices, keep it at the bottom of this func
	ld.Bin, ld.RBin, ld.WBin, ld.TBin = lrs.Bins(bins)

	// warning: reorders lrs by value, it is no longer in time order!
	sort.Sort(lrs)

	// populates the percentiles map with another pass over lrs
	ld.Pcntl = percentiles(lrs)

	// Find the index of the 1st percentile, then build bins on the slice from 0 to P1
	p1idx := ld.Pcntl[1].Idx
	p1lrs := lrs[:p1idx]
	ld.P1Bin, ld.P1RBin, ld.P1WBin, ld.P1TBin = p1lrs.Bins(bins)

	// Find the index of the 99th percentile, then build bins on the slice from P99 to the last sample
	p99idx := ld.Pcntl[99].Idx
	p99lrs := lrs[p99idx:]
	ld.P99Bin, ld.P99RBin, ld.P99WBin, ld.P99TBin = p99lrs.Bins(bins)

	return
}

// expects lrs to be pre-sorted
func percentiles(lrs LogRecs) LogPcntl {
	out := make(LogPcntl, 102)

	pctl_idx := func(pc float64) int {
		idx := math.Floor(float64(len(lrs)) * (pc / 100))
		out := int(idx)
		return out
	}

	var i float64
	for i = 1; i <= 99; i += 1 {
		idx := pctl_idx(i)
		out[i] = lrs[idx]
		out[i].Idx = uint32(idx) // track index for building P1/P99 bins
	}

	out[99.9] = lrs[pctl_idx(99.9)]
	out[99.99] = lrs[pctl_idx(99.99)]
	out[99.999] = lrs[pctl_idx(99.999)]

	return out
}

func (lrs LogRecs) Bins(bins int) (all, read, write, trim LogBin) {
	all = NewLogBin(bins)   // all-IO bin
	read = NewLogBin(bins)  // reads bin
	write = NewLogBin(bins) // writes bin
	trim = NewLogBin(bins)  // trims bin

	// one pass to count each direction of IO
	var all_count, read_count, write_count, trim_count int
	for _, lr := range lrs {
		all_count++
		if lr.Ddir == 0 {
			read_count++
		} else if lr.Ddir == 1 {
			write_count++
		} else if lr.Ddir == 2 {
			trim_count++
		}
	}

	var arec, rrec, wrec, trec int // next record index
	var acnt, rcnt, wcnt, tcnt int // bucket counter

	// bucketSize() returns floored (count / buckets) with error checking
	// when these buckets are filled, they are aggregated then reused
	abkt := make(LogRecs, bucketSize(bins, all_count))
	rbkt := make(LogRecs, bucketSize(bins, read_count))
	wbkt := make(LogRecs, bucketSize(bins, write_count))
	tbkt := make(LogRecs, bucketSize(bins, trim_count))

	// TODO: document this algorithm for one pass bucket filling
	for i, lr := range lrs {
		// check bounds on each call: when bin_width * bins < record_count
		// care must be taken to not call updateBucket which may try to write
		// a bin entry beyond the end of the slice
		if arec < len(rbkt) && acnt < bins {
			arec, acnt = abkt.updateBucket(arec, acnt, all, lrs, i)
		}

		if lr.Ddir == 0 && rrec < len(rbkt) && rcnt < bins {
			rrec, rcnt = rbkt.updateBucket(rrec, rcnt, read, lrs, i)
		} else if lr.Ddir == 1 && wrec < len(wbkt) && wcnt < bins {
			wrec, wcnt = wbkt.updateBucket(wrec, wcnt, write, lrs, i)
		} else if lr.Ddir == 2 && trec < len(tbkt) && tcnt < bins {
			trec, tcnt = tbkt.updateBucket(trec, tcnt, trim, lrs, i)
		}
	}

	return
}

// compute the bucket size
// always take the floor of the value; a short bucket will have
// a count that is skewed by the sample count. Since bucket count
// is usually pretty small, 10-100, it's better to simply drop samples
// that would be in a complete bucket rather than keeping them and having
// all my graphs take a dive on the last sample
func bucketSize(buckets int, available int) int {
	if available < buckets {
		return 0
	}

	return int(math.Floor(float64(available) / float64(buckets)))
}

// Adds the value to the bucket at index bktidx using the LogRec at lrs[lridx].
// When full, summarized into smry[smry_idx]. Returns updated index values for
// use in the next iteration.
//
// It is safe to use the same bucket on each iteration to save allocation.
//
// bktidx: current bucket index
// hgidx: current bin index
// bin: destination bin (list) - written to!
// lrs: source data slice
// lridx: current index into the source data slice
// Returns: (new bucket index, new bin index)
func (bucket LogRecs) updateBucket(bktidx int, hgidx int, bin LogBin, lrs LogRecs, lridx int) (int, int) {
	// if the bucket is len 0, there weren't enough samples
	if len(bucket) == 0 {
		return 0, 0
	}

	// at max_lridx all buckets are full; remaining records must be dropped
	max_lridx := (len(lrs) - (len(lrs) % len(bucket))) - 1

	// add the current LogRec to the bucket
	if lridx <= max_lridx {
		bucket[bktidx] = lrs[lridx]
	}

	// advance the bucket index, stay on the same summary index
	if bktidx < len(bucket)-1 && lridx < max_lridx {
		return bktidx + 1, hgidx
		// discard leftover records after filling all buckets: see bucketSize()
	} else if lridx > max_lridx {
		return bktidx, hgidx
		// bucket is full or end of data, sum it & advance to the next bin entry
	} else if bktidx == len(bucket)-1 || lridx == max_lridx {
		hs := LogSmry{}

		// finding max/min ts by indices would usually work, but the backing LogRecs
		// is sorted in place at times, so be safe and do it the hard way
		hs.MinTs = math.MaxUint32

		// bucket is a static size, but at the end of a dataset there might not
		// be enough samples to fill it, so always use `bslice` instead of `bucket` here
		// which is shortened as needed
		bslice := bucket[0:]
		if lridx == len(lrs)-1 {
			bslice = bucket[0 : bktidx+1]
		}

		// count and sum up all entries, find min/max timestamp
		for _, lr := range bslice {
			hs.Sum += uint64(lr.Val)
			hs.Count++

			if lr.Time > hs.MaxTs {
				hs.MaxTs = lr.Time
			}

			if lr.Time < hs.MinTs {
				hs.MinTs = lr.Time
			}
		}

		// get the median/p50 and average values
		hs.Median = uint64(bslice[(len(bslice)-1)/2].Val)
		hs.Average = float64(hs.Sum) / float64(hs.Count)

		// add up the squares of each value's delta from average
		var dsum float64
		for _, lr := range bslice {
			dsum += math.Pow(float64(lr.Val)-hs.Average, 2)
		}

		// finish computing the standard deviation
		variance := dsum / float64(hs.Count)
		hs.Stdev = math.Sqrt(variance)

		// sort by value then get the percentiles
		sort.Sort(bslice)
		hs.Pcntl = percentiles(bslice)

		// save to the bin summary
		bin[hgidx] = &hs

		// bucket is now summed & stored at hgidx, reset the bucket index to 0
		return 0, hgidx + 1
	}

	panic("BUG! Unreachable!")
}

// JSON doesn't officially support anything but strings as keys
// so the floats have to be converted with this handler.
func (lp LogPcntl) MarshalJSON() ([]byte, error) {
	jsonFmt := "\"%g\": {\"time\": %d, \"value\": %d}%s"
	max := fmt.Sprintf(jsonFmt, math.MaxFloat64, math.MaxInt32, math.MaxInt32, ",")
	buf := make([]byte, len(lp)*len(max)+4)

	// copy the keys to a list for sorting so they're in order in the output
	count := 0
	keys := make([]float64, len(lp))
	for key, _ := range lp {
		keys[count] = key
		count++
	}

	sort.Float64s(keys)

	sep := ","
	buf[0] = '{'
	bufidx := 1
	for i, key := range keys {
		if i == len(keys)-1 {
			sep = ""
		}
		out := fmt.Sprintf(jsonFmt, key, lp[key].Time, lp[key].Val, sep)
		outbytes := []byte(out)
		bufidx += copy(buf[bufidx:bufidx+len(outbytes)], outbytes)
	}
	buf[bufidx] = '}'

	// buf was overallocated generously, return a precisely sized array
	out := make([]byte, bufidx+1)
	copy(out, buf[0:bufidx+1])

	return out, nil
}

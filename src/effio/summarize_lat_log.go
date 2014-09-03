package effio

import (
	"fmt"
	"math"
	"sort"
)

// Latency Record: The 4 fields from fio's latency logs and an index cache
// This is where most of the memory goes
type LatRec struct {
	Time uint32 `json:"time"`  // time offset from beginning of fio run
	Val  uint32 `json:"value"` // latency value in usec
	Ddir uint8  `json:"-"`     // 0 = read, 1 = write, 2 = trim
	Bsz  uint16 `json:"-"`     // block size
	Idx  uint32 `json:"-"`     // save the original index in LatRecs
}
type LatPcntl map[float64]*LatRec // .MarshalJSON() at EOF
type LatRecs []*LatRec

// sort interface impl, sorts by value for indexing percentiles
func (p LatRecs) Len() int           { return len(p) }
func (p LatRecs) Less(i, j int) bool { return p[i].Val < p[j].Val }
func (p LatRecs) Swap(i, j int)      { p[i].Val, p[j].Val = p[j].Val, p[i].Val }

// Latency Bucket Summary: a handful of useful values for each bucket in
// the LatHgram.
type LatSmry struct {
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
	Pcntl   LatPcntl `json:"percentiles"`
}
type LatHgram []*LatSmry

func NewLatHgram(size int) LatHgram {
	lhg := make(LatHgram, size)
	for i, _ := range lhg {
		lhg[i] = &LatSmry{}
	}
	return lhg
}

type LatSummaries struct {
	// the fio command used to generate the file
	FioCommand FioCommand `json:"fio_command"`
	// data from the output of fio --output=json
	FioJsonData FioJsonData `json:"fio_data"`
	// the global summary
	Summary LatSmry `json:"summary"`
	// all 99 percentiles + 99.9, 99.99, and 99.999%
	Pcntl LatPcntl `json:"percentiles"`
	// histogram across all samples, then by io direction
	Histogram  LatHgram `json:"histogram"`       // hgram of all records
	RHistogram LatHgram `json:"read_histogram"`  // hgram of all read ops
	WHistogram LatHgram `json:"write_histogram"` // hgram of all write ops
	THistogram LatHgram `json:"trim_histogram"`  // hgram of all trim ops
	// capture outliers by preserving full resolution for metrics <P1 and >P99
	P1Histogram   LatHgram `json:"p1_histogram"`        // hgram of records with values < P1
	P1RHistogram  LatHgram `json:"p1_read_histogram"`   // hgram <P1 / read
	P1WHistogram  LatHgram `json:"p1_write_histogram"`  // hgram <P1 / write
	P1THistogram  LatHgram `json:"p1_trim_histogram"`   // hgram <P1 / trim
	P99Histogram  LatHgram `json:"p99_histogram"`       // hgram of records with values > P99
	P99RHistogram LatHgram `json:"p99_read_histogram"`  // hgram >P99 / read
	P99WHistogram LatHgram `json:"p99_write_histogram"` // hgram >P99 / write
	P99THistogram LatHgram `json:"p99_trim_histogram"`  // hgram >P99 / trim
}

// Summarizes the LatRecs data into a LatSmry.
// First argument is the number of samples to put in the summaries.
// Second argument is the number of buckets in the histograms.
// This does all the work in 3 passes, the first getting avg/min/max.
// Then the values are sorted to access the percentiles by index.
// The final pass computes the standard deviation, which requires the average
// from the first pass.
func (lrs LatRecs) Summarize(histogram_size int) (ld LatSummaries) {
	if histogram_size > len(lrs) {
		histogram_size = len(lrs)
	}

	smry := LatSmry{
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
	ld.Histogram, ld.RHistogram, ld.WHistogram, ld.THistogram = lrs.Histograms(histogram_size)

	// warning: reorders lrs by value, it is no longer in time order!
	sort.Sort(lrs)

	// populates the percentiles map with another pass over lrs
	ld.Pcntl = percentiles(lrs)

	// Find the index of the 1st percentile, then build histograms on the slice from 0 to P1
	p1idx := ld.Pcntl[1].Idx
	p1lrs := lrs[:p1idx]
	ld.P1Histogram, ld.P1RHistogram, ld.P1WHistogram, ld.P1THistogram = p1lrs.Histograms(histogram_size)

	// Find the index of the 99th percentile, then build histograms on the slice from P99 to the last sample
	p99idx := ld.Pcntl[99].Idx
	p99lrs := lrs[p99idx:]
	ld.P99Histogram, ld.P99RHistogram, ld.P99WHistogram, ld.P99THistogram = p99lrs.Histograms(histogram_size)

	return
}

func (lrs LatRecs) Histograms(histogram_size int) (all, read, write, trim LatHgram) {
	all = NewLatHgram(histogram_size)   // all-IO histogram
	read = NewLatHgram(histogram_size)  // reads histogram
	write = NewLatHgram(histogram_size) // writes histogram
	trim = NewLatHgram(histogram_size)  // trims histogram

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

	// bucketSize() returns rounded (count / buckets) with error checking
	abkt := make(LatRecs, bucketSize(histogram_size, all_count))
	rbkt := make(LatRecs, bucketSize(histogram_size, read_count))
	wbkt := make(LatRecs, bucketSize(histogram_size, write_count))
	tbkt := make(LatRecs, bucketSize(histogram_size, trim_count))

	// TODO: document this algorithm for one pass bucket filling
	for i, lr := range lrs {
		arec, acnt = abkt.updateBucket(arec, acnt, all, lrs, i)

		if lr.Ddir == 0 {
			rrec, rcnt = rbkt.updateBucket(rrec, rcnt, read, lrs, i)
		} else if lr.Ddir == 1 {
			wrec, wcnt = wbkt.updateBucket(wrec, wcnt, write, lrs, i)
		} else if lr.Ddir == 2 {
			trec, tcnt = tbkt.updateBucket(trec, tcnt, trim, lrs, i)
		}
	}

	return
}

// expects lrs to be pre-sorted
func percentiles(lrs LatRecs) LatPcntl {
	out := make(LatPcntl, 102)

	pctl_idx := func(pc float64) int {
		idx := math.Floor(float64(len(lrs)) * (pc / 100))
		out := int(idx)
		return out
	}

	var i float64
	for i = 1; i <= 99; i += 1 {
		idx := pctl_idx(i)
		out[i] = lrs[idx]
		out[i].Idx = uint32(idx) // track index for building P1/P99 histograms
	}

	out[99.9] = lrs[pctl_idx(99.9)]
	out[99.99] = lrs[pctl_idx(99.99)]
	out[99.999] = lrs[pctl_idx(99.999)]

	return out
}

// compute the bucket size, default to 1 if less than histogram_size
func bucketSize(buckets int, available int) int {
	if buckets < available {
		return int(math.Ceil(float64(available) / float64(buckets)))
	} else if available == 0 {
		return 0
	} else {
		panic(fmt.Sprintf("Sample count (%d) < Bucket count (%d): returning bucket size of 1.\n", available, buckets))
	}
}

// Adds the value to the bucket at index bktidx using the LatRec at lrs[lridx].
// When full, summarized into smry[smry_idx]. Returns updated index values for
// use in the next iteration.
//
// It is safe to use the same bucket on each iteration to save allocation.
//
// bktidx: current bucket index
// hgidx: current histogram index
// hgram: histogram (list) - written to!
// lrs: source data slice
// lridx: current index into the source data slice
// Returns: (new bucket index, new histogram index)
func (bucket LatRecs) updateBucket(bktidx int, hgidx int, hgram LatHgram, lrs LatRecs, lridx int) (int, int) {
	// add the current LatRec to the bucket
	bucket[bktidx] = lrs[lridx]

	// advance the bucket index, stay on the same summary index
	if bktidx < len(bucket)-1 && lridx < len(lrs)-1 {
		return bktidx + 1, hgidx
		// bucket is full or end of data, sum it & advance to the next histogram entry
	} else {
		hs := LatSmry{}

		// finding max/min ts by indices would usually work, but the backing LatRecs
		// is sorted in place at times, so be safe and do it the hard way
		hs.MinTs = math.MaxUint32

		// bucket is a static size, but at the end of a dataset there might not
		// be enough samples to fill it, so always use `bslice` instead of `bucket` here
		// which is shortened as needed
		bslice := bucket[0:]
		if lridx == len(lrs)-1 {
			bslice = bucket[0:bktidx]
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
		hs.Median = uint64(bslice[len(bslice)/2].Val)
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

		// save to the histogram summary
		hgram[hgidx] = &hs

		// bucket is now summed & stored at hgidx, reset the bucket index to 0
		return 0, hgidx + 1
	}
}

// JSON doesn't officially support anything but strings as keys
// so the floats have to be converted with this handler.
func (lp LatPcntl) MarshalJSON() ([]byte, error) {
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

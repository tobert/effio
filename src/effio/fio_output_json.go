package effio

// methods for loading & wrangling fio's JSON output
// License: Apache 2.0

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
)

type FioJsonHistogram map[float64]float64

// three kinds of latency, slat, clat, lat all with the same fields
type FioJsonLatency struct {
	Min        float64          `json:"min"`
	Max        float64          `json:"max"`
	Mean       float64          `json:"mean"`
	Stdev      float64          `json:"stdev"`
	Percentile FioJsonHistogram `json:"percentile"`
}

// same data for Mixed/Read/Write/Trim, which are present depends on
// how fio was run
type FioJsonJobStats struct {
	IoBytes   int             `json:"io_bytes"`
	Bandwidth float64         `json:"bw"`
	BwMin     float64         `json:"bw_min"`
	BwMax     float64         `json:"bw_max"`
	BwAgg     float64         `json:"bw_agg"`
	BwMean    float64         `json:"bw_mean"`
	BwStdev   float64         `json:"bw_dev"`
	Iops      int             `json:"iops"`
	Runtime   int             `json:"runtime"`
	Slat      *FioJsonLatency `json:"slat"`
	Clat      *FioJsonLatency `json:"clat"`
	Lat       *FioJsonLatency `json:"lat"`
}

// each fio session can have multiple jobs, each job is reported
// in an array called client_stats

type FioJsonJob struct {
	Name              string           `json:"jobname"`
	Description       string           `json:"desc"`
	Groupid           int              `json:"groupid"`
	Error             int              `json:"error"`
	Mixed             *FioJsonJobStats `json:"mixed"` // fio config dependent
	Read              *FioJsonJobStats `json:"read"`  // fio config dependent
	Write             *FioJsonJobStats `json:"write"` // fio config dependent
	Trim              *FioJsonJobStats `json:"trim"`  // fio config dependent
	UsrCpu            float64          `json:"usr_cpu"`
	SysCpu            float64          `json:"sys_cpu"`
	ContextSwitches   int              `json:"ctx"`
	MajorFaults       int              `json:"majf"`
	MinorFaults       int              `json:"minf"`
	IODepthLevel      FioJsonHistogram `json:"iodepth_level"`
	LatencyUsec       FioJsonHistogram `json:"latency_us"`
	LatencyMsec       FioJsonHistogram `json:"latency_ms"`
	LatencyDepth      int              `json:"latency_depth"`
	LatencyTarget     int              `json:"latency_target"`
	LatencyPercentile float64          `json:"latency_percentile"`
	LatencyWindow     int              `json:"latency_window"`
	Hostname          string           `json:"hostname"`
	Port              int              `json:"port"`
}

type FioJsonDiskUtil struct {
	Name        string  `json:"name"`
	ReadIos     int     `json:"read_ios"`
	WriteIos    int     `json:"write_ios"`
	ReadMerges  int     `json:"read_merges"`
	WriteMerges int     `json:"write_merges"`
	ReadTicks   int     `json:"read_ticks"`
	WriteTicks  int     `json:"write_ticks"`
	InQueue     int     `json:"in_queue"`
	Util        float64 `json:"util"`
}

type FioJsonData struct {
	Filename      string            `json:"filename"`
	FioVersion    string            `json:"fio version"`
	HeaderGarbage string            `json:"garbage"`
	Jobs          []FioJsonJob      `json:"jobs"`
	DiskUtil      []FioJsonDiskUtil `json:"disk_util"`
}

func LoadFioJsonData(filename string) (fdata FioJsonData) {
	dataBytes, err := ioutil.ReadFile(filename)

	if os.IsNotExist(err) {
		log.Fatalf("Could not read file %s: %s", filename, err)
	}

	// data loaded OK
	fdata.Filename = filename

	// fio writes a bunch of crap out to the output file before the JSON so for
	// now do the easy thing and find the first { after a \n and call it good
	offset := bytes.Index(dataBytes, []byte("\n{"))
	// bytes.Index will return -1 for not found, in which case we assume that it
	// been trimmed from the input file and start at index 0
	if offset == -1 {
		offset = 0
	}

	err = json.Unmarshal(dataBytes[offset:], &fdata)
	if err != nil {
		log.Fatalf("Could not parse JSON: %s", err)
	}

	fdata.HeaderGarbage = string(dataBytes[0:offset])

	return
}

// some of the bucket keys are in the form ">=50.00" which of course
// cannot be unmarshaled into a number, so clean that up before trying
func (hst *FioJsonHistogram) UnmarshalJSON(data []byte) error {
	hststr := make(map[string]float64, len(data)/8)

	err := json.Unmarshal(data, &hststr)
	if err != nil {
		return err
	}

	out := make(FioJsonHistogram, len(hststr))

	for k, v := range hststr {
		// remove the ">=" fio puts in some of the keys
		cleaned := strings.TrimPrefix(k, ">=")
		fkey, _ := strconv.ParseFloat(cleaned, 64)
		out[fkey] = v
	}

	hst = &out

	return nil
}

// JSON doesn't officially support anything but strings as keys
// so the floats have to be converted with this handler.
func (hst FioJsonHistogram) MarshalJSON() ([]byte, error) {
	started := false
	buf := make([]byte, 8192) // lazy
	sep := ""
	buf[0] = '{'
	bufidx := 1
	for key, val := range hst {
		if started {
			sep = ","
		} else {
			started = true
		}
		out := []byte(fmt.Sprintf("\"%g\":%g%s", key, val, sep))
		bufidx += copy(buf[bufidx:bufidx+len(out)], out)
	}
	buf[bufidx] = '}'

	return buf[0 : bufidx+1], nil
}

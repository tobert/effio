package effio

// effio/fio_json.go - methods for loading & wrangling fio's JSON output
// License: Apache 2.0

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
)

func (fdata *FioData) LoadJSON(filename string) {
	dataBytes, err := ioutil.ReadFile(filename)

	if os.IsNotExist(err) {
		log.Fatal("Could not read file %s: %s", filename, err)
	}

	// fio writes a bunch of crap out to the output file before the JSON
	// so for now do the easy thing and find the first { after a \n
	// and call it good enough
	offset := bytes.Index(dataBytes, []byte("\n{"))

	err = json.Unmarshal(dataBytes[offset:], &fdata)
	if err != nil {
		log.Fatal("Could parse JSON: %s", err)
	}

	fdata.HeaderGarbage = string(dataBytes[0:offset])

	// now go over the maps of string => float64 and fix them up to be float64 => float64
	for _, cs := range fdata.ClientStats {
		cs.IODepthLevel = cs.IODepthLevelStr.cleanKeys()
		cs.LatencyUsec = cs.LatencyUsecStr.cleanKeys()
		cs.LatencyMsec = cs.LatencyMsecStr.cleanKeys()

		// might be worth checking for valid combinations someday, but in practice this works OK
		cs.Mixed.cleanHistograms()
		cs.Read.cleanHistograms()
		cs.Write.cleanHistograms()
		cs.Trim.cleanHistograms()
	}
}

// the same 3 fields exist in Read/Write/Mixed/Trim
func (js *JobStats) cleanHistograms() {
	// JSON might not contain this field
	if js == nil {
		return
	}
	js.Lat.Percentile = js.Lat.PercentileStr.cleanKeys()
	js.Clat.Percentile = js.Clat.PercentileStr.cleanKeys()
	js.Slat.Percentile = js.Slat.PercentileStr.cleanKeys()
}

// some of the bucket keys are in the form ">=50.00" which of course
// cannot be unmarshaled into a number, so clean that up before trying
func (hst HistogramStr) cleanKeys() Histogram {
	// JSON might not contain this field
	if hst == nil {
		return nil
	}

	out := make(Histogram, len(hst))

	for k, v := range hst {
		// remove the ">=" fio puts in some of the keys
		cleaned := strings.TrimPrefix(k, ">=")
		fkey, _ := strconv.ParseFloat(cleaned, 64)
		out[fkey] = v
	}

	return out
}

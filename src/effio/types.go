package effio

// effio/types.go - types to represent all of fio's output in Go
// License: Apache 2.0

// some of the distributions in fio's output in all output modes
// have keys that come in the form of ">=64". Let the JSON decoder
// load those keys as strings in HistogramStr and clean them up
// with a second pass writing into Histogram types
type Histogram map[float64]float64
type HistogramStr map[string]float64

// three kinds of latency, slat, clat, lat all with the same fields
type Latency struct {
	Min           float64      `json:"min"`
	Max           float64      `json:"max"`
	Mean          float64      `json:"mean"`
	Stdev         float64      `json:"stdev"`
	PercentileStr HistogramStr `json:"percentile"`
	Percentile    Histogram
}

// same data for Mixed/Read/Write/Trim, which are present depends on
// how fio was run
type JobStats struct {
	IoBytes   int      `json:"io_bytes"`
	Bandwidth float64  `json:"bw"`
	BwMin     float64  `json:"bw_min"`
	BwMax     float64  `json:"bw_max"`
	BwAgg     float64  `json:"bw_agg"`
	BwMean    float64  `json:"bw_mean"`
	BwStdev   float64  `json:"bw_dev"`
	Iops      int      `json:"iops"`
	Runtime   int      `json:"runtime"`
	Slat      *Latency `json:"slat"`
	Clat      *Latency `json:"clat"`
	Lat       *Latency `json:"lat"`
}

// each fio session can have multiple jobs, each job is reported
// in an array called client_stats
type ClientStat struct {
	Jobname           string       `json:"jobname"`
	Groupid           int          `json:"groupid"`
	Error             int          `json:"error"`
	Mixed             *JobStats    `json:"mixed"` // fio config dependent
	Read              *JobStats    `json:"read"`  // fio config dependent
	Write             *JobStats    `json:"write"` // fio config dependent
	Trim              *JobStats    `json:"trim"`  // fio config dependent
	UsrCpu            float64      `json:"usr_cpu"`
	SysCpu            float64      `json:"sys_cpu"`
	ContextSwitches   int          `json:"ctx"`
	MajorFaults       int          `json:"majf"`
	MinorFaults       int          `json:"minf"`
	IODepthLevelStr   HistogramStr `json:"iodepth_level"`
	LatencyUsecStr    HistogramStr `json:"latency_us"`
	LatencyMsecStr    HistogramStr `json:"latency_ms"`
	IODepthLevel      Histogram    // filled in after unmarshal
	LatencyUsec       Histogram    // filled in after unmarshal
	LatencyMsec       Histogram    // filled in after unmarshal
	LatencyDepth      int          `json:"latency_depth"`
	LatencyTarget     int          `json:"latency_target"`
	LatencyPercentile float64      `json:"latency_percentile"`
	LatencyWindow     int          `json:"latency_window"`
	Hostname          string       `json:"hostname"`
	Port              int          `json:"port"`
}

type FioData struct {
	Filename      string
	FioVersion    string        `json:"fio version"`
	HeaderGarbage string        `json:"garbage"`
	ClientStats   []ClientStat  `json:"client_stats"`
	DiskUtil      []interface{} `json:"disk_util"` // unused for now
}

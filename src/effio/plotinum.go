package effio

import (
	"code.google.com/p/plotinum/plot"
	"code.google.com/p/plotinum/plotter"
	"code.google.com/p/plotinum/plotutil"
	"code.google.com/p/plotinum/vg"
	"fmt"
	"log"
	"os"
	"path"
)

type Group struct {
	Name     string
	Tests    Tests
	Grouping *Grouping
}
type Groups map[string]*Group
type Grouping struct {
	Name      string // group name, e.g. "by_fio", "by_media"
	SuitePath string // root of the suite, e.g. /home/atobey/src/effio/suites/-id/
	OutPath   string // writing final graphs in this directory
	Groups    Groups // e.g. "samsung_840_read_latency" => [ t1, t2, ... ]
	Suite     *Suite // parent test suite
}

// suite_path must be a fully-qualitifed path or Chdirs will fail and crash
func (suite *Suite) GraphAll(suite_path string, out_path string) {
	// various groupings/pivots that will be graphed
	by_fio := NewGrouping("by_fio_conf", out_path, suite_path, suite)
	by_dev := NewGrouping("by_device", out_path, suite_path, suite)
	by_mda := NewGrouping("by_media", out_path, suite_path, suite)
	by_tst := NewGrouping("by_test", out_path, suite_path, suite)
	all := []Grouping{by_fio, by_dev, by_mda, by_tst}

	// assign tests to groups
	for _, test := range suite.Tests {
		by_fio.AppendGroup(test.FioConfTmpl.Name, test) // e.g. "read_latency_512" => [ t1, t9, .. ]
		by_dev.AppendGroup(test.Device.Name, test)      // e.g. "fusionio_iodriveii" => [ t3, t7, ...]
		by_mda.AppendGroup(test.Device.Media, test)     // e.g. "MLC" => [t1, t6, ...]
		by_tst.AppendGroup(test.Name, test)             // ends up 1:1 name => [t1]
	}

	// generate a latency logfile size graph for every group
	for _, gg := range all {
		for _, g := range gg.Groups {
			g.barFileSizes()
		}
	}

	// load all data into memory
	// will be rather large but probably OK on a 16GB machine
	for _, test := range suite.Tests {
		// LatRec implements the plotinum interfaces Valuer, etc. and can be used directly
		recs := LoadCSV(test.LatLogPath(suite_path))

		// but plotinum chokes on huge files so reduce those down if they're over 1e5 entries
		// TODO: this could be a runtime flag, since plotinum does finish with huge sample
		// sizes but it takes 5-10 minutes per graph at 8e6 samples.
		if len(recs) > 1000 {
			test.LatRecs = recs.Histogram(1000)
		} else {
			test.LatRecs = recs
		}
	}

	for _, gg := range all {
		for _, g := range gg.Groups {
			g.scatterPlot()
		}
	}
}

func NewGrouping(name string, out_path string, suite_path string, suite *Suite) Grouping {
	mbrs := make(Groups)
	return Grouping{name, suite_path, out_path, mbrs, suite}
}

func (gg *Grouping) AppendGroup(key string, test *Test) {
	if g, ok := gg.Groups[key]; ok {
		g.Tests = append(gg.Groups[key].Tests, test)
	} else {
		gg.Groups[key] = &Group{key, Tests{test}, gg}
	}
}

func (g *Group) scatterPlot() {
	p, err := plot.New()
	if err != nil {
		log.Fatalf("Error creating new plot: %s\n", err)
	}

	// TODO: human names for test groups
	p.Title.Text = fmt.Sprintf("Latency Distribution: %s", g.Name)
	p.X.Label.Text = "Time Offset"
	p.Y.Label.Text = "Latency (usec)"
	p.Add(plotter.NewGrid())
	p.Legend.Top = true

	for i, test := range g.Tests {
		sp, err := plotter.NewScatter(test.LatRecs)
		if err != nil {
			log.Fatalf("Failed to create new scatter plot for test %s: %s\n", test.Name, err)
		}
		sp.GlyphStyle.Color = plotutil.Color(i)
		p.Add(sp)
		p.Legend.Add(test.Name, sp)
	}

	g.saveGraph(p, "scatter")
}

// draws a bar graph displaying the sizes of the lat_lat.log files across
// all tests
// TODO: figure out how to make the bar width respond to the graph width
func (g *Group) barFileSizes() {
	sizes := make([]int64, len(g.Tests))
	for i, test := range g.Tests {
		fi, err := os.Stat(test.LatLogPath(g.Grouping.SuitePath))
		if err != nil {
			sizes[i] = 0
			continue
		}
		sizes[i] = fi.Size()
	}

	p, err := plot.New()
	if err != nil {
		log.Fatalf("Error creating new plot: %s\n", err)
	}

	p.Title.Text = fmt.Sprintf("Latency Log Sizes: %s", g.Name)
	p.X.Label.Text = "Device + Test"
	p.Y.Label.Text = "Bytes"
	p.Legend.Top = true
	p.Add(plotter.NewGrid())

	// plotinum doesn't offer a way to draw one group of bars
	// with different colors, so each bar is a group with an offset
	var bw float64 = 20.0
	var count float64 = 0
	for i, test := range g.Tests {
		if sizes[i] == 0 {
			continue
		}

		val := plotter.Values{float64(sizes[i])}
		chart, err := plotter.NewBarChart(val, vg.Points(bw))
		if err != nil {
			log.Fatalf("Error adding bar to plot: %s\n", err)
		}

		chart.Color = plotutil.Color(i)
		chart.Offset = vg.Points(count * bw)

		p.Add(chart)
		p.Legend.Add(test.Name, chart)

		count += 1
	}

	p.X.Min = 0
	p.X.Max = float64(count + 1)

	g.saveGraph(p, "bar-log-size")
}

// e.g. suites/-id/-out/scatter-by_dev-random-read-512b.jpg
func (g *Group) saveGraph(p *plot.Plot, name string) {
	fname := fmt.Sprintf("%s-%s-%s.png", name, g.Grouping.Name, g.Name)
	fpath := path.Join(g.Grouping.OutPath, fname)
	err := p.Save(10, 10, fpath)
	if err != nil {
		log.Fatalf("Failed to save %s: %s\n", fpath, err)
	}
	log.Printf("saved graph: '%s'\n", fpath)
}

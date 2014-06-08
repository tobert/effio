package effio

import (
	"code.google.com/p/plotinum/plot"
	"code.google.com/p/plotinum/plotter"
	"code.google.com/p/plotinum/vg"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"runtime"
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
	Groups    Groups `json:"-"` // e.g. "samsung_840_read_latency" => [ t1, t2, ... ]
	Suite     *Suite `json:"-"` // parent test suite
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

	for _, gg := range all {
		for _, g := range gg.Groups {
			// generate a latency logfile size graph for every group
			g.barFileSizes()

			// load the CSV on demand
			// at one point this cached loaded tests between runs, but as long
			// as plotinum is taking minutes to generate graphs with lots of data
			// points, the file loading doesn't cost enough to matter
			for _, test := range g.Tests {
				test.LatRecs = LoadCSV(test.LatLogPath(g.Grouping.SuitePath))
				test.LatData = test.LatRecs.Summarize()

				// release the memory used by loading the raw data then force a GC
				// otherwise some of the CSV files easily OOM a 16G machine
				test.LatRecs = nil
				runtime.GC()

				test.LatData.WriteFiles(gg.OutPath, fmt.Sprintf("%s-%s", gg.Name, g.Name))
			}

			// generate output
			g.scatterPlot(true)
			g.scatterPlot(false)
			g.barChart(true)
			g.barChart(false)

			// write metadata for the group/grouping as json
			g.writeJson()
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

func (g *Group) barChart(logscale bool) {
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
	w := vg.Points(20)

	for i, test := range g.Tests {
		bars, err := plotter.NewBarChart(test.LatData.Histogram, w)
		if err != nil {
			log.Fatalf("Failed to create new barchart for test %s: %s\n", test.Name, err)
		}
		bars.Color = CustomColors[i]
		p.Add(bars)
		p.Legend.Add(fmt.Sprintf("read: %s ", test.Device.Name), bars)
	}

	if logscale {
		p.Y.Scale = plot.LogScale
		p.Y.Label.Text = "Latency (usec log(10))"
		g.saveGraph(p, "scatter-logscale")
	} else {
		g.saveGraph(p, "scatter")
	}
}

func (g *Group) scatterPlot(logscale bool) {
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
		if len(test.LatData.RRecSm) > 0 {
			// reads get circles
			rsp, err := plotter.NewScatter(test.LatData.RRecSm)
			if err != nil {
				log.Fatalf("Failed to create new scatter plot for test %s: %s\n", test.Name, err)
			}
			rsp.Shape = plot.CircleGlyph{}
			rsp.GlyphStyle.Color = CustomColors[i]
			p.Add(rsp)
			p.Legend.Add(fmt.Sprintf("read: %s ", test.Device.Name), rsp)
		}

		if len(test.LatData.WRecSm) > 0 {
			// writes get pyramids, same color
			wsp, err := plotter.NewScatter(test.LatData.WRecSm)
			if err != nil {
				log.Fatalf("Failed to create new scatter plot for test %s: %s\n", test.Name, err)
			}
			wsp.Shape = plot.PyramidGlyph{}
			wsp.GlyphStyle.Color = CustomColors[i]
			p.Add(wsp)
			p.Legend.Add(fmt.Sprintf("write: %s ", test.Device.Name), wsp)
		}
	}

	if logscale {
		p.Y.Scale = plot.LogScale
		p.Y.Label.Text = "Latency (usec log(10))"
		g.saveGraph(p, "scatter-logscale")
	} else {
		g.saveGraph(p, "scatter")
	}
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

		chart.Color = CustomColors[i]
		chart.Offset = vg.Points(count * bw)

		p.Add(chart)
		p.Legend.Add(test.Name, chart)

		count += 1
	}

	p.X.Min = 0
	p.X.Max = float64(count + 1)

	g.saveGraph(p, "bar-log-size")
}

func (g *Group) writeJson() {
	fname := fmt.Sprintf("group-%s-%s.json", g.Grouping.Name, g.Name)
	outfile := path.Join(g.Grouping.OutPath, fname)

	js, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		log.Fatalf("Failed to encode group data as JSON: %s\n", err)
	}
	js = append(js, byte('\n'))

	err = ioutil.WriteFile(outfile, js, 0644)
	if err != nil {
		log.Fatalf("Failed to write group JSON data file '%s': %s\n", outfile, err)
	}
}

// e.g. suites/-id/-out/scatter-by_dev-random-read-512b.jpg
func (g *Group) saveGraph(p *plot.Plot, name string) {
	fname := fmt.Sprintf("%s-%s-%s.svg", name, g.Grouping.Name, g.Name)
	fpath := path.Join(g.Grouping.OutPath, fname)
	err := p.Save(12, 8, fpath)
	if err != nil {
		log.Fatalf("Failed to save %s: %s\n", fpath, err)
	}
	log.Printf("saved graph: '%s'\n", fpath)
}

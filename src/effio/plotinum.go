package effio

import (
	"code.google.com/p/plotinum/plot"
	"code.google.com/p/plotinum/plotter"
	"code.google.com/p/plotinum/vg"
	"code.google.com/p/plotinum/plotutil"
	"fmt"
	"log"
	"os"
	"path"
)

// suite_path must be a fully-qualitifed path or Chdirs will fail and crash
func (suite *Suite) Graph(suite_path string, outdir string) {
	// load all data into memory
	// will be rather large but probably OK on a 16GB machine
	recs := make([]LatRecs, len(suite.Tests))

	for i, test := range suite.Tests {
		recs[i] = LoadCSV(test.LatLogPath(suite_path))
		if len(recs[i]) < 200 {
			log.Printf("Empty/truncated logfile. Skipping rendering of %s\n", test.Name)
			continue
		}
	}
}

// draws a bar graph displaying the sizes of the lat_lat.log files across
// all tests
// TODO: figure out how to make the bar width respond to the graph width
func (suite *Suite) GraphSizes(suite_path string, outdir string) {
	sizes := make([]int64, len(suite.Tests))
	for i, test := range suite.Tests {
		fi, err := os.Stat(test.LatLogPath(suite_path))
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

	p.Title.Text = fmt.Sprintf("Latency Log Sizes")
	p.X.Label.Text = "Device + Test"
	p.Y.Label.Text = "Bytes"
	p.Legend.Top = true
	p.Add(plotter.NewGrid())

	// plotinum doesn't offer a way to draw one group of bars
	// with different colors, so each bar is a group with an offset
	var bw float64 = 20.0
	var count float64 = 0
	for i, test := range suite.Tests {
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

	fname := path.Join(outdir, "lat-log-sizes.png")
	err = p.Save(10, 10, fname)
	if err != nil {
		log.Fatalf("Failed to save %s: %s\n", fname, err)
	}
	log.Printf("saved sizes graph to '%s'\n", fname)
}

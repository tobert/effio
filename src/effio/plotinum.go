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

// Returns a fully-qualified path to the lat_lat.log CSV file
func (test *Test) LatLogPath(suite_path string) string {
	tpath := path.Join(suite_path, test.Dir)
	// TODO: check validity with stat

	// fio insists on adding the _lat.log and I can't find an option to disable it
	return path.Join(tpath, fmt.Sprintf("%s_lat.log", test.FioLatLog))
}

func (test *Test) GraphAll(suite_path string) {
	latlog := LoadCSV(test.LatLogPath(suite_path))
	// latlog is huge on fast devices, trim it down so plotinum doesn't freak out
	//func (lrs LatRecs) Histogram(sz int) (out LatRecs) {
	hgram := latlog.Histogram(200)
	test.LineGraph(hgram, "Time", "Latency (usec)")
}

func (test *Test) LineGraph(data plotter.XYer, xlabel string, ylabel string) {
	p, err := plot.New()
	if err != nil {
		log.Fatalf("Error creating new plot: %s\n", err)
	}

	p.Title.Text = fmt.Sprintf("Latency %s", test.Device)
	p.X.Label.Text = xlabel
	p.Y.Label.Text = ylabel

	p.Add(plotter.NewGrid())

	log.Printf("Adding data with l, err := plotter.NewLine(data)\n")
	l, err := plotter.NewLine(data)
	if err != nil {
		log.Fatalf("Error adding line to plot: %s\n", err)
	}
	l.LineStyle.Width = vg.Points(1)

	p.Add(l)

	log.Printf("saving graph to linegraph.png\n")
	err = p.Save(10, 10, "linegraph.png")
	if err != nil {
		log.Fatalf("Failed to save linegraph.png: %s\n", err)
	}
	log.Printf("all done\n")
}

package effio

import (
	"code.google.com/p/plotinum/plot"
	"code.google.com/p/plotinum/plotter"
	"code.google.com/p/plotinum/vg"
	//	"code.google.com/p/plotinum/plotutil"
	"fmt"
	"log"
	"os"
	"path"
)

func (suite *Suite) Graph(spath string, outdir string) {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Could not get working directory: %s\n", err)
	}

	for _, test := range suite.Tests {
		log.Printf("Graphing test %s ...\n", test.Name)
		test.GraphAll(path.Join(wd, spath))
	}
}

func (test *Test) GraphAll(spath string) {
	tpath := path.Join(spath, test.Dir)
	err := os.Chdir(tpath)
	if err != nil {
		log.Fatalf("Could not chdir to '%s': %s\n", tpath, err)
	}

	// fio insists on adding the _lat.log and I can't find an option to disable it
	latlog := LoadCSV(fmt.Sprintf("%s_lat.log", test.FioLatLog))
	if len(latlog) < 200 {
		log.Printf("Empty/truncated logfile. Skipping rendering of %s\n", test.Name)
		return
	}
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

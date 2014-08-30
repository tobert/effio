package effio

import (
	"encoding/hex"
	"hash/fnv"
	"log"
	"path"
	"sort"
)

type GraphSuiteCmd struct {
	SuiteCmd
	OutFlag  string
	ListFlag bool
}

func (cmd *Cmd) ToGraphSuiteCmd() GraphSuiteCmd {
	sc := cmd.ToSuiteCmd()
	gsc := GraphSuiteCmd{sc, "", false}
	gsc.FlagSet.StringVar(&gsc.OutFlag, "out", "all", "name of the directory that will contain graphs: -path/-id/-out")
	gsc.FlagSet.BoolVar(&gsc.ListFlag, "list", false, "print a list of included tests and exit without processing")
	return gsc
}

func (gsc *GraphSuiteCmd) Run() {
	if !gsc.FlagSet.Parsed() {
		err := gsc.FlagSet.Parse(gsc.Args)
		if err != nil {
			log.Fatalf("BUG: flag parsing failed: %s\n", err)
		}
	}

	gsc.ParseArgs()

	suite := gsc.LoadSuite()

	outdir := path.Join(gsc.WorkPath, gsc.PathFlag, gsc.IdFlag, gsc.OutFlag)

	// if incl/excl are used and an 'out' name isn't specified, make one
	// based on a hash of all names in the test so it's consistent and automatic
	if gsc.OutFlag == "all" && (len(gsc.InclFlag) > 0 || len(gsc.ExclFlag) > 0) {
		sort.Sort(suite.Tests) // sort to ensure the hash is as consistent as possible
		hash := fnv.New64()
		for _, test := range suite.Tests {
			hash.Write([]byte(test.Name)) // docs: never returns an error
		}
		name := hex.EncodeToString(hash.Sum(nil))
		outdir = path.Join(gsc.WorkPath, gsc.PathFlag, gsc.IdFlag, name)
		log.Printf("output will be written to '%s'\n", outdir)
	}

	if gsc.ListFlag {
		for _, test := range suite.Tests {
			log.Println(test.Name)
		}
		return
	}

	suite.GraphAll(gsc.SuitePath, outdir)
}

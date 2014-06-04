package effio

/*
 * effio provides ways to build a suite of fio tests & devices, combine them
 * then generate reports / graphs.
 *
 * The fio config file name is stripped of its path and .fio extension to be
 * used in composite test names. A test against the above device with
 * "rand_512b_write_iops.fio" will be named
 * "rand_512b_write_iops-samsung_840_pro_256".
 */

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"time"
)

// a test suite has a global id that is also used as a directory name
type Suite struct {
	Id        string
	Created   time.Time // time the test was generated / run
	EffioCmd  []string  // os.Args() of the effio command used to generate the suite
	SuiteJson string
	Tests     Tests
}

// NewSuite returns an initialized Suite with the given
// id and the Created field set to the current time.
func NewSuite(id string) Suite {
	now := time.Now()
	fname := path.Join(id, "suite.json")
	return Suite{id, now, os.Args, fname, []Test{}}
}

// LoadSuiteJson loads a suite from JSON. Argument is a path to a
// JSON file that has a complete suite's information in it.
func LoadSuiteJson(spath string) (suite Suite) {
	data, err := ioutil.ReadFile(spath)
	if err != nil {
		log.Fatalf("Could not read suite JSON file '%s': %s", spath, err)
	}

	err = json.Unmarshal(data, &suite)
	if err != nil {
		log.Fatalf("Could not parse suite JSON in file '%s': %s", spath, err)
	}

	return suite
}

// Run the whole suite one at a time letting fio write its output into
// the suite directories. Repeated runs will overwrite files; behavior
// is dependent on what fio does with existing files for now.
func (suite *Suite) Run(spath string) {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Could not get working directory: %s\n", err)
	}

	for _, test := range suite.Tests {
		log.Printf("Running test %s in directory %s ...\n", test.Name, test.Dir)
		test.Run(path.Join(wd, spath))
	}
}

// Populate the test suite with the (cartesian) product of
// Devices x FioConfTmpls to get all combinations.
// This does not modify the filesystem.
func (suite *Suite) Populate(dl Devices, ftl FioConfTmpls) {
	for _, tp := range ftl {
		for _, dev := range dl {
			if dev.Ignore {
				continue
			}

			// I suppose these conventions could be defined higher up in the call stack
			// but this makes things a little easier to modify down the road.
			testName := fmt.Sprintf("%s-%s", dev.Name, tp.Name)
			testDir := path.Join(suite.Id, testName)
			args := []string{"--output-format=json", "--output=output.json", "config.fio"}

			// fio adds _$type.log to log file names so only provide the base name
			test := Test{
				Name:        testName,
				Dir:         testDir,
				FioTestName: tp.Name,
				FioArgs:     args,
				FioFile:     "config.fio",
				FioJson:     "output.json",
				FioBWLog:    "bw",
				FioLatLog:   "lat",
				FioIopsLog:  "iops",
				TestJson:    "test.json",
				CmdFile:     "run.sh",
				FioConfTmpl: tp,
				Device:      dev,
				Suite:       suite,
			}

			suite.Tests = append(suite.Tests, test)
		}
	}
}

// WriteAll(path) writes a suite out to a set of directories and files.
func (suite *Suite) WriteAll(basePath string) {
	suite.mkdirAll(basePath)

	suite.WriteSuiteJson(basePath)

	for _, test := range suite.Tests {
		test.WriteFioFile(basePath)
		test.WriteTestJson(basePath)
		test.WriteCmdFile(basePath)
	}
}

// WriteSuiteJson dumps the suite data structure to a JSON file. This
// file is used by some effio subcommands, such as run_suite and various
// reports.
// <basePath>/<suite id>/suite.json
func (suite *Suite) WriteSuiteJson(basePath string) {
	outfile := path.Join(basePath, suite.SuiteJson)

	js, err := json.MarshalIndent(suite, "", "  ")
	if err != nil {
		log.Fatalf("Failed to encode suite data as JSON: %s\n", err)
	}

	// MarshalIndent does not follow the final brace with a newline
	js = append(js, byte('\n'))

	err = ioutil.WriteFile(outfile, js, 0644)
	if err != nil {
		log.Fatalf("Failed to write suite JSON data file '%s': %s\n", outfile, err)
	}
}

// mkdirAll(path) creates the directory structure of a test suite
// under directory 'path'. This must be called before the Write*()
// methods or they will fail. It only makes sense to call this after
// Populate().
func (suite *Suite) mkdirAll(basePath string) {
	sdir := path.Join(basePath, suite.Id)

	for _, t := range suite.Tests {
		tdir := path.Join(sdir, t.Name)
		err := os.MkdirAll(tdir, 0755)
		if err != nil {
			log.Fatalf("Failed to create test directory '%s': %s\n", tdir, err)
		}
	}
}

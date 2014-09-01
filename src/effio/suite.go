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
	"path/filepath"
	"time"
)

type Suite struct {
	Name        string      `json:"name"`         // a name given to the suite on the command line
	Path        string      `json:"path"`         // path for writing benchmark data out
	StartTS     time.Time   `json:"start_ts"`     // time the suite was started
	EndTS       time.Time   `json:"end_ts"`       // time the suite finished
	EffioCmd    []string    `json:"effio_cmd"`    // os.Args() of the effio command used
	SuiteJson   string      `json:"suite_json"`   // metadata about the suite of tests
	FioCommands FioCommands `json:"fio_commands"` // fio commands run/to be run
}

// NewSuite returns an initialized Suite with the given
// id and the Created field set to the current time.
func NewSuite(name string, pathArg string) Suite {
	absPath, err := filepath.Abs(pathArg)
	if err != nil {
		log.Fatalf("Could not determine the absolute path of '%s': %s\n", pathArg, err)
	}

	spath := path.Join(absPath, name)
	fname := path.Join(absPath, name, "suite.json")

	return Suite{
		Name:        name,
		Path:        spath,
		StartTS:     time.Now(),
		EffioCmd:    os.Args,
		SuiteJson:   fname,
		FioCommands: FioCommands{},
	}
}

// Run the whole suite one at a time letting fio write its output into
// the suite directories. Repeated runs will overwrite files; behavior
// is dependent on what fio does with existing files for now.
func (suite *Suite) Run(rerun bool) {
	for _, fcmd := range suite.FioCommands {
		// rerun = true means all benchmarks get re-run
		// when false, only benchmarks with missing or empty output.json get run
		if !rerun {
			if fcmd.FioJsonSize() > 0 {
				continue
			}
		}
		fcmd.Run()
	}
}

// Populate the suite with the (cartesian) product of Devices x FioConfTmpls
// to get all combinations (in memory).
func (suite *Suite) Populate(dl Devices, ftl FioConfTmpls) {
	for _, tp := range ftl {
		for _, dev := range dl {
			if dev.Ignore {
				continue
			}

			// These conventions could be defined higher up in the call stack
			// but this makes things a little easier to modify down the road.
			fcmdName := fmt.Sprintf("%s-%s", dev.Name, tp.Name)
			fcmdPath := path.Join(suite.Path, fcmdName)
			args := []string{"--output-format=json", "--output=output.json", "config.fio"}

			// fio adds _$type.log to log file names so only provide the base name
			fcmd := FioCommand{
				Name:        fcmdName,
				Path:        fcmdPath,
				FioArgs:     args,
				FioFile:     "config.fio",
				FioJson:     "output.json",
				FioBWLog:    "bw",
				FioLatLog:   "lat",
				FioIopsLog:  "iops",
				CmdJson:     "test.json",
				CmdScript:   "run.sh",
				FioConfTmpl: tp,
				Device:      dev,
			}

			suite.FioCommands = append(suite.FioCommands, &fcmd)
		}
	}
}

// WriteAll() writes a suite out to a set of directories and files.
func (suite *Suite) WriteAll() {
	suite.mkdirAll()

	suite.WriteSuiteJson()

	for _, fcmd := range suite.FioCommands {
		fcmd.WriteFioConf()
		fcmd.WriteFcmdJson()
		fcmd.WriteCmdScript()
	}
}

// WriteSuiteJson() dumps the suite data structure to a JSON file. This
// file is used by some effio subcommands, such as run_suite and various
// reports.
// <suite path>/<suite id>/suite.json
func (suite *Suite) WriteSuiteJson() {
	js, err := json.MarshalIndent(suite, "", "  ")
	if err != nil {
		log.Fatalf("Failed to encode suite data as JSON: %s\n", err)
	}

	// MarshalIndent does not follow the final brace with a newline
	js = append(js, byte('\n'))

	err = ioutil.WriteFile(suite.SuiteJson, js, 0644)
	if err != nil {
		log.Fatalf("Failed to write suite JSON data file '%s': %s\n", suite.SuiteJson, err)
	}
}

// mkdirAll() creates the directory structure of a test suite
// under directory 'path'. This must be called before the Write*()
// methods or they will fail. It only makes sense to call this after
// Populate().
func (suite *Suite) mkdirAll() {
	sdir := path.Join(suite.Path, suite.Name)

	for _, fcmd := range suite.FioCommands {
		fcdir := path.Join(sdir, fcmd.Name)
		err := os.MkdirAll(fcdir, 0755)
		if err != nil {
			log.Fatalf("Failed to create directory '%s': %s\n", fcdir, err)
		}
	}
}

package effio

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"
)

// test data generated on the fly based on info above
type Test struct {
	Name        string      // name to be used in tests, files, etc.
	Dir         string      // directory for writing configs, logs, etc.
	FioTestName string      // name of the fio test (without the device name)
	FioArgs     []string    // the arguments to the fio command for the test
	FioFile     string      // generated fio config file name
	FioJson     string      // generated fio json output file name
	FioBWLog    string      // filename for the bandwidth log
	FioLatLog   string      // filename for the latency log
	FioIopsLog  string      // filename for the iops log
	TestJson    string      // dump the test data (this struct) to this file
	CmdFile     string      // write the exact fio command used to this file
	FioConfTmpl FioConfTmpl // template info struct
	Device      Device      // device info struct
	Suite       *Suite      `json:"-"`
}

// has a sort interface impl near EOF, sorts by test.Name
type Tests []Test

func (test *Test) Run(spath string) {
	tpath := path.Join(spath, test.Dir)

	err := os.Chdir(tpath)
	if err != nil {
		log.Fatalf("Could not chdir to '%s': %s\n", tpath, err)
	}

	fioPath, err := exec.LookPath("fio")
	if err != nil {
		log.Fatalf("Could not locate an fio command in PATH: %s\n", err)
	}

	cmd := exec.Command(fioPath, test.FioArgs...)
	before := time.Now()

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatalf("Could not run '%s %s': %s\n", fioPath, strings.Join(test.FioArgs, " "))
	}

	// grab stderr in case something goes wrong
	// TODO: switch this to io.Copy?
	errors, err := ioutil.ReadAll(stderr)
	if err != nil {
		log.Fatal(err)
	}

	err = cmd.Wait()

	// TODO: figure out if this is worth recording and record it
	elapsed := time.Since(before)

	// it might be OK to let 1 fio command out of a suite fail?
	if err != nil {
		log.Printf(string(errors))
		log.Fatalf("Command '%s %s' failed: %s\n", fioPath, strings.Join(test.FioArgs, " "), err)
	}
	log.Printf("Elapsed: %s\n", elapsed)
}

// DumpFioFile writes the fio configuration file.
// <basePath>/<suite id>/<generated test name>/config.fio
func (test *Test) WriteFioFile(basePath string) {
	outfile := path.Join(basePath, test.Dir, test.FioFile)

	fd, err := os.OpenFile(outfile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalf("Failed to create fio config file '%s': %s\n", outfile, err)
	}
	defer fd.Close()

	err = test.FioConfTmpl.tmpl.Execute(fd, test)
	if err != nil {
		log.Fatalf("Template execution failed for '%s': %s\n", outfile, err)
	}
}

// WriteTestJson dumps the suite data structure to a JSON file for posterity (and debugging).
// <basePath>/<suite id>/<generated test name>/test.json
func (test *Test) WriteTestJson(basePath string) {
	outfile := path.Join(basePath, test.Dir, test.TestJson)

	js, err := json.MarshalIndent(test, "", "  ")
	if err != nil {
		log.Fatalf("Failed to encode test data as JSON: %s\n", err)
	}

	// MarshalIndent does not follow the final brace with a newline
	js = append(js, byte('\n'))

	err = ioutil.WriteFile(outfile, js, 0644)
	if err != nil {
		log.Fatalf("Failed to write test JSON data file '%s': %s\n", outfile, err)
	}
}

// WriteCmdFile writes the command to a file as a mini shell script.
// <basePath>/<suite id>/<test name>/run.sh
func (test *Test) WriteCmdFile(basePath string) {
	outfile := path.Join(basePath, test.Dir, test.CmdFile)

	fd, err := os.OpenFile(outfile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		log.Fatalf("Failed to create command file '%s': %s\n", outfile, err)
	}
	defer fd.Close()

	// just use 'fio' if it isn't found on the path
	fioPath, err := exec.LookPath("fio")
	if err != nil {
		fioPath = "fio"
	}
	fmt.Fprintf(fd, "#!/bin/bash -x\n%s %s\n", fioPath, strings.Join(test.FioArgs, " "))
}

// Returns a fully-qualified path to the lat_lat.log CSV file
func (test *Test) LatLogPath(suite_path string) string {
	tpath := path.Join(suite_path, test.Dir)
	// TODO: check validity with stat

	// fio insists on adding the _lat.log and I can't find an option to disable it
	return path.Join(tpath, fmt.Sprintf("%s_lat.log", test.FioLatLog))
}

// implement the sort for Tests
func (t Tests) Len() int {
	return len(t)
}

func (t Tests) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

// sort by test Name lexically
func (t Tests) Less(i, j int) bool {
	return t[i].Name < t[j].Name
}

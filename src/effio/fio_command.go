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

// FioCommand: everything to do with running fio benchmarks.
// The goal is to capture every detail of how the benchmark was generated
// and eventually run so it can be exported with all results.
type FioCommand struct {
	Name        string      `json:"name"`          // name to be used in commands, files, etc.
	Path        string      `json:"path"`          // directory for writing configs, logs, etc.
	MinTs       time.Time   `json:"min_ts"`        // timestamp right before starting fio
	MaxTs       time.Time   `json:"max_ts"`        // timestamp right after the process exits
	FioArgs     []string    `json:"fio_args"`      // the arguments to the executed fio command
	FioFile     string      `json:"fio_file"`      // generated fio config file name
	FioJson     string      `json:"fio_json"`      // generated fio json output file name
	FioBWLog    string      `json:"fio_bw_log"`    // filename for the bandwidth log
	FioLatLog   string      `json:"fio_lat_log"`   // filename for the latency log
	FioIopsLog  string      `json:"fio_iops_log"`  // filename for the iops log
	CmdJson     string      `json:"command_json"`  // dump of the fio command data (this struct)
	CmdScript   string      `json:"command_sh"`    // a shell script with the fio command in it
	FioConfTmpl FioConfTmpl `json:"fio_conf_tmpl"` // template info struct
	Device      Device      `json:"device"`        // device info struct
	Suite       *Suite      `json:"-"`             // don't serialize to JSON
}

// FioCommands: A sortable list of FioCommand
type FioCommands []*FioCommand

func (fcs FioCommands) Len() int           { return len(fcs) }
func (fcs FioCommands) Swap(i, j int)      { fcs[i], fcs[j] = fcs[j], fcs[i] }
func (fcs FioCommands) Less(i, j int) bool { return fcs[i].Name < fcs[j].Name }

// Run() an fio benchmark
func (fcmd *FioCommand) Run() {
	err := os.Chdir(fcmd.Path)
	if err != nil {
		log.Fatalf("Could not chdir to output path '%s': %s\n", fcmd.Path, err)
	}

	fioPath, err := exec.LookPath("fio")
	if err != nil {
		log.Fatalf("Could not locate an fio command in PATH: %s\n", err)
	}

	unmount := false
	if fcmd.Device.Device != "" && fcmd.Device.Mountpoint != "" {
		err := fcmd.Device.Mount()
		if err != nil {
			log.Printf(fcmd.Device.ToJson())
			log.Fatalf("Could not mount device '%s': %s\n", fcmd.Device.Name, err)
		}
		unmount = true
	}

	// start collecting data from /proc/diskstats in a goroutine
	stopstats := CollectDiskstats(path.Join(fcmd.Path, "diskstats.csv"), fcmd.Device)

	// set up the process
	cmd := exec.Command(fioPath, fcmd.FioArgs...)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}

	// start running the process
	if err := cmd.Start(); err != nil {
		log.Fatalf("Could not run '%s %s': %s\n", fioPath, strings.Join(fcmd.FioArgs, " "))
	}

	// grab stderr in case something goes wrong
	errors, err := ioutil.ReadAll(stderr)
	if err != nil {
		log.Fatal(err)
	}

	// blocking wait for the process to exit
	err = cmd.Wait()

	// stop the diskstats collection goroutine
	close(stopstats)

	if unmount {
		err = fcmd.Device.Umount()
		if err != nil {
			log.Printf(fcmd.Device.ToJson())
			log.Fatalf("Could not unmount device '%s': %s\n", fcmd.Device.Name)
		}
	}

	// it might be OK to let 1 fio command out of a suite fail?
	if err != nil {
		log.Printf(string(errors))
		log.Fatalf("Command '%s %s' failed: %s\n", fioPath, strings.Join(fcmd.FioArgs, " "), err)
	}
}

// WriteFioConf() writes the fio configuration file.
// <-path path>/<suite.Name>/<generated command name>/config.fio
func (fcmd *FioCommand) WriteFioConf() {
	outfile := path.Join(fcmd.Path, fcmd.FioFile)

	fd, err := os.OpenFile(outfile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalf("Failed to create fio config file '%s': %s\n", outfile, err)
	}
	defer fd.Close()

	err = fcmd.FioConfTmpl.tmpl.Execute(fd, fcmd)
	if err != nil {
		log.Fatalf("Template execution failed for '%s': %s\n", outfile, err)
	}
}

// WriteFcmdJson() dumps the fio command data to a JSON file
// <-path path>/<suite.Name>/<fcmd.Name>/command.json
func (fcmd *FioCommand) WriteFcmdJson() {
	outfile := path.Join(fcmd.Path, fcmd.CmdJson)

	js, err := json.MarshalIndent(fcmd, "", "  ")
	if err != nil {
		log.Fatalf("Failed to encode command data as JSON: %s\n", err)
	}

	// MarshalIndent does not follow the final brace with a newline
	js = append(js, byte('\n'))

	err = ioutil.WriteFile(outfile, js, 0644)
	if err != nil {
		log.Fatalf("Failed to write command JSON data file '%s': %s\n", outfile, err)
	}
}

func LoadFioCommandJson(filename string) (out FioCommand) {
	dataBytes, err := ioutil.ReadFile(filename)
	if os.IsNotExist(err) {
		log.Fatalf("Could not read file %s: %s", filename, err)
	}

	err = json.Unmarshal(dataBytes, &out)
	if err != nil {
		log.Fatalf("Could not parse FioCommand JSON: %s", err)
	}

	return out
}

// WriteCmdScript() writes the command to a file as a mini shell script.
// <-path path>/<suite.Name>/<fcmd.Name>/run.sh
func (fcmd *FioCommand) WriteCmdScript() {
	outfile := path.Join(fcmd.Path, fcmd.CmdScript)

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
	fmt.Fprintf(fd, "#!/bin/bash -x\n%s %s\n", fioPath, strings.Join(fcmd.FioArgs, " "))
}

// Returns a fully-qualified path to the lat_lat.log CSV file
func (fcmd *FioCommand) LatLogPath() string {
	// fio insists on adding the _lat.log and I can't find an option to disable it
	return path.Join(fcmd.Path, fmt.Sprintf("%s_lat.log", fcmd.FioLatLog))
}

// get the size of the latency log, return 0 on errors (e.g. missing)
func (fcmd *FioCommand) LatLogSize() int64 {
	fi, err := os.Stat(fcmd.LatLogPath())
	if err != nil {
		return 0
	}
	return fi.Size()
}

// FioJsonSize() gets the size of output.json, returns 0 on errors
func (fcmd *FioCommand) FioJsonSize() int64 {
	fpath := path.Join(fcmd.Path, fcmd.FioJson)
	fi, err := os.Stat(fpath)
	if err != nil {
		return 0
	}
	return fi.Size()
}

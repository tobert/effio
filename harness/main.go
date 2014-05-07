package main

/*
 * A harness for running fio on a bunch of devices consistently. I use this
 * to run a series of tests on my test bench. Processes a directory containing
 * fio configs, generating a new directory with filled-out configs for fio and
 * the graphing tools in this package.
 *
 * The JSON device file is an array of devices and some relevant data. e.g.
 * [
 *   {
 *     "name":       "samsung_840_pro_256",
 *     "device":     "/dev/disk/by-id/ata-Samsung_SSD_840_PRO_Series_S1ATNEAD541857W",
 *     "mountpoint": "/mnt/sda",
 *     "filesystem": "ext4",
 *     "brand":      "Samsung",
 *     "series":     "840 PRO",
 *     "capacity":   256060514304,
 *     "rotational": false,
 *     "transport":  "SATA",
 *     "hba":        "AHCI",
 *     "media":      "MLC",
 *     "blocksize":  512
 *   }
 * ]
 *
 * Descriptions:
 *   name:       manually assigned, will be used in file names!
 *   device:     always use the /dev/disk/by-id/ path
 *   mountpoint: location where the filesystem is mounted
 *   filesystem: ext4, xfs, zfs, btrfs, ntfs-3g
 *   brand:      Samsung, Fusion IO, I/O Switch Tech, Seagate, Western Digital, etc.
 *   series:     "840 PRO",
 *   capacity:   `blockdev --getsize64 /dev/sda`
 *   rotational: false for SSD, true for HDD, true if device contains any HDD
 *   transport:  SATA, SAS, PCIe, MDRAID, iSCSI, virtio
 *   hba:        ioMemory, AHCI, SAS3004, USB3, mixed (for MDRAID)
 *   media:      MLC, Iron (for HDDs), TLC, SLC, Hybrid (SSHD)
 *   blocksize:  `blockdev --getpbsz /dev/sda`
 *
 * The fio config file name is stripped of its path and .fio extension to be
 * used in composite test names. A test against the above device with
 * "rand_512b_write_iops.fio" will be named
 * "rand_512b_write_iops-samsung_840_pro_256".
 *
 * 2014-05-07T19:21:00Z/
 *   rand_512b_write_iops-samsung_840_pro_256/
 *     config.fio - the fio configuration file
 *     harness.json  - a dump of all data used to generate the test
 *     output.json  - json output from fio --output-format=json
 *     run.sh     - the exact command used to run fio
 */

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// loaded from JSON per the -devs flag or defaults to <hostname>.json
type Device struct {
	Name       string
	Device     string
	Mountpoint string
	Filesystem string
	Brand      string
	Series     string
	Capacity   int64
	Rotational bool
	Transport  string
	HBA        string
	Media      string
	BlockSize  int
}

type DeviceList []Device

// fio has some built-in variable expansion but I want to save the artifacts
// to a git repo
type FioTmpl struct {
	Filename string             // full path to the source file
	Name     string             // used to generate output filenames
	Template *template.Template `json:"-"`
}

type FioTmplList []FioTmpl

// test data generated on the fly based on info above
type Test struct {
	Name     string    // name to be used in tests, files, etc.
	Time     time.Time // time the test was generated / run
	Dir      string    // directory for writing configs, logs, etc.
	FioCmd   string    // the fio command to execute to run the test
	BWLog    string    // filename for the bandwidth log
	LatLog   string    // filename for the latency log
	IopsLog  string    // filename for the iops log
	FioFile  string    // generated fio config file name
	DataFile string    // dump the test data (this struct) to this file
	CmdFile  string    // write the exact fio command used to this file
	Template FioTmpl   // template info struct
	Dev      Device    // device info struct
}

type TestSuite []Test

// command-line flags (global)
var devJsonFlag, confDirFlag, outPathFlag string

func init() {
	// the default device filename is <hostname>.json
	devfile, err := os.Hostname()
	if err != nil {
		devfile = "devices"
	}
	devfile = fmt.Sprintf("./%s.json", devfile)

	flag.StringVar(&devJsonFlag, "devs", devfile, "A JSON file containing device metadata")
	flag.StringVar(&confDirFlag, "conf", "./fio_configs", "A directroy containing fio config templates")
	flag.StringVar(&outPathFlag, "out", "./conf", "Where to write the generated output, must be writeable")
}

func main() {
	flag.Parse()

	// change any relative paths to absolute paths
	devJson := required(filepath.Abs(devJsonFlag))
	confDir := required(filepath.Abs(confDirFlag))
	outPath := required(filepath.Abs(outPathFlag))

	// load device data from json
	devs := loadDevJson(devJson)

	// load the fio config templates into memory
	templates := loadFioTmpl(confDir)

	// build up a test suite of devs x templates
	suite := buildTestSuite(devs, templates, outPath)

	// write out all the files
	for _, test := range suite {
		err := os.MkdirAll(test.Dir, 0755)
		if err != nil {
			log.Fatalf("Failed to create test directory '%s': %s\n", test.Dir, err)
		}

		test.writeFioFile()
		test.writeDataFile()
		test.writeCmdFile()
	}

	// TODO: run the tests!
}

// for each device/fio config combination, create a config file in
// a new directory named <iso8601 date>/<name> with one directory
// per test so fio can be excuted in those directories, keeping
// data generated along side the configs
func buildTestSuite(devs DeviceList, templates FioTmplList, outPath string) (suite TestSuite) {
	// get the current time once and use it for the whole suite
	now := time.Now()

	for _, tp := range templates {
		for _, dev := range devs {
			testName := fmt.Sprintf("%s-%s", dev.Name, tp.Name)
			testDir := path.Join(outPath, testName)
			fioJson := path.Join(outPath, "output.json")
			fioConf := path.Join(testDir, "config.fio")
			cmd := fmt.Sprintf("fio --output-format=json --output=%s %s", fioJson, fioConf)

			// fio adds _$type.log to log file names so only provide the base name
			test := Test{
				Name:     testName,
				Time:     now,
				Dir:      testDir,
				FioCmd:   cmd,
				FioFile:  fioConf,
				BWLog:    path.Join(testDir, "bw"),
				LatLog:   path.Join(testDir, "lat"),
				IopsLog:  path.Join(testDir, "iops"),
				DataFile: path.Join(testDir, "harness.json"),
				CmdFile:  path.Join(testDir, "run.sh"),
				Template: tp,
				Dev:      dev,
			}

			suite = append(suite, test)
		}
	}

	return suite
}

func (test *Test) writeFioFile() {
	fd, err := os.OpenFile(test.FioFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalf("Failed to create fio config file '%s': %s\n", test.FioFile, err)
	}
	defer fd.Close()

	err = test.Template.Template.Execute(fd, test)
	if err != nil {
		log.Fatalf("Template execution failed: %s\n", err)
	}
}

func (test *Test) writeDataFile() {
	js, err := json.MarshalIndent(test, "", "  ")
	if err != nil {
		log.Fatalf("Failed to encode test data as JSON: %s\n", err)
	}

	// MarshalIndent does not follow the final brace with a newline
	js = append(js, byte('\n'))

	err = ioutil.WriteFile(test.DataFile, js, 0644)
	if err != nil {
		log.Fatalf("Failed to write JSON data file '%s': %s\n", test.DataFile, err)
	}
}

func (test *Test) writeCmdFile() {
	fd, err := os.OpenFile(test.CmdFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		log.Fatalf("Failed to create command file '%s': %s\n", test.CmdFile, err)
	}
	defer fd.Close()

	fmt.Fprintf(fd, "#!/bin/bash -x\n%s\n", test.FioCmd)
}

func loadDevJson(fname string) (devs DeviceList) {
	mdbuf, err := ioutil.ReadFile(fname)
	if err != nil {
		log.Fatalf("Could not read '%s': %s\n", fname, err)
	}
	err = json.Unmarshal(mdbuf, &devs)
	if err != nil {
		log.Fatalf("Could not parse JSON: %s\n", err)
	}

	return devs
}

func loadFioTmpl(dir string) (out FioTmplList) {
	visitor := func(fpath string, f os.FileInfo, err error) error {
		if err != nil {
			log.Fatalf("Encountered an error while loading fio config '%s': %s", fpath, err)
		}

		fname := path.Base(fpath)
		ext := path.Ext(fname)
		if ext == ".fio" {
			data, err := ioutil.ReadFile(fpath)
			if err != nil {
				log.Fatalf("Could not read fio config '%s': %s", fpath, err)
			}

			// remove the .fio to get the base filename to use as a generic name string
			name := strings.TrimSuffix(fname, ext)
			tmpl := template.Must(template.New(name).Parse(string(data)))

			out = append(out, FioTmpl{fpath, name, tmpl})
		}

		return nil
	}

	err := filepath.Walk(dir, visitor)
	if err != nil {
		log.Fatalf("Could not load configs in '%s': %s", dir, err)
	}

	return out
}

func required(data string, err error) string {
	if err != nil {
		log.Fatalf("BUG: Required operation failed with error: %s\n", err)
	}

	return data
}

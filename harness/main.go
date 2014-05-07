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
 *     "name":       "samsung-840-pro-256",
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
	Filename string // full path to the file
	Basename string // used to generate output filenames
	Template string // raw template data
}

type FioTmplList []FioTmpl

// test data generated on the fly based on info above
type Test struct {
	Name     string    // name to be used in tests, files, etc.
	Time     time.Time // time the test was generated / run
	Dir      string    // directory for writing configs, logs, etc.
	BWLog    string    // filename for the bandwidth log
	LatLog   string    // filename for the latency log
	IopsLog  string    // filename for the iops log
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
	devfile = fmt.Sprintf("%s.json", devfile)

	flag.StringVar(&devJsonFlag, "devs", devfile, "A JSON file containing device metadata")
	flag.StringVar(&confDirFlag, "conf", "./fio_configs", "A directroy containing fio config templates")
	flag.StringVar(&outPathFlag, "out", "./conf", "Where to write the generated output, must be writeable")
}

func main() {
	flag.Parse()

	// get the current time once and use it for the whole suite
	now := time.Now()

	// load device data from json
	devs := loadDevJson(devJsonFlag)

	// load the fio config templates into memory
	templates := loadFioTmpl(confDirFlag)

	// for each device/fio config combination, create a config file in
	// a new directory named <name>-<iso8601 date> with one directory
	// per test so fio can be excuted in those directories, keeping
	// data generated along side the configs
	suite := TestSuite{}
	for _, tp := range templates {
		for _, dev := range devs {
			testName := fmt.Sprintf("%s-%s", dev.Name, tp.Basename)
			testDir := path.Join(outPathFlag, testName)

			// fio adds _$type.log to log file names so only provide the base name
			test := Test{
				Name:     testName,
				Time:     now,
				Dir:      testDir,
				BWLog:    fmt.Sprintf("bw-%s", testName),
				LatLog:   fmt.Sprintf("lat-%s", testName),
				IopsLog:  fmt.Sprintf("iops-%s", testName),
				Template: tp,
				Dev:      dev,
			}
			suite = append(suite, test)
		}
	}

	// TODO: generate fio config, shell command, dump test to JSON
	fmt.Printf("%v\n", suite)

	// TODO: run the tests!
}

func loadDevJson(fname string) (devs DeviceList) {
	mdbuf, err := ioutil.ReadFile(devJsonFlag)
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

			out = append(out, FioTmpl{fpath, fname, string(data)})
		}

		return nil
	}

	err := filepath.Walk(dir, visitor)
	if err != nil {
		log.Fatalf("Could not load configs in '%s': %s", dir, err)
	}

	return out
}

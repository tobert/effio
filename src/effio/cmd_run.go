package effio

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// effio run -dev <file.json> -fio <dir> -path <dir>
func (cmd *Cmd) RunSuite() {
	// the default device filename is <hostname>.json
	devfile, err := os.Hostname()
	if err != nil {
		devfile = "devices"
	}
	devfile = fmt.Sprintf("conf/machines/%s.json", devfile)

	// parse subcommand arguments
	var nameFlag, devFlag, fioFlag, pathFlag string
	var dryrunFlag, rerunFlag bool
	fs := cmd.FlagSet
	fs.StringVar(&nameFlag, "name", "", "name of the suite")
	fs.StringVar(&devFlag, "dev", devfile, "JSON file containing device metadata")
	fs.StringVar(&fioFlag, "fio", "conf/fio/default", "directory containing fio config templates")
	fs.StringVar(&pathFlag, "path", "./suites/", "where to write out generated data")
	fs.BoolVar(&dryrunFlag, "dryrun", false, "only generate metadata, without running fio")
	fs.BoolVar(&rerunFlag, "rerun", false, "only rerun fio benchmarks with missing or empty output.json")
	fs.Parse(cmd.Args)

	if nameFlag == "" {
		fs.Usage()
	}

	// load device data from json
	devs := LoadDevicesFile(mustAbs(devFlag))

	// load the fio config templates into memory
	templates := LoadFioConfDir(mustAbs(fioFlag))

	// use an absolute directory for pathFlag
	outPath := mustAbs(pathFlag)

	// build up a test suite of devs x templates
	suite := NewSuite(nameFlag, outPath)
	suite.Populate(devs, templates)
	suite.WriteAll()

	if !dryrunFlag {
		suite.Run(rerunFlag)
	}
}

// mustAbs change any relative path to an absolute path
// any error from filepath.Abs is considered fatal
func mustAbs(p string) string {
	out, err := filepath.Abs(p)
	if err != nil {
		log.Fatalf("BUG: Required operation failed with error: %s\n", err)
	}

	return out
}

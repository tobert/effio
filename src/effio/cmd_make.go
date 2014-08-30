package effio

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

type MakeCmd struct {
	SuiteCmd
	DevFlag string
	FioFlag string
}

func (cmd *Cmd) ToMakeCmd() (mc MakeCmd) {

}

// effio make -dev <file.json> -fio <dir> -path <dir>
func (mc *MakeCmd) Run() {
	// the default device filename is conf/machines/<hostname>.json
	devfile, err := os.Hostname()
	if err != nil {
		devfile = "devices"
	}
	devfile = fmt.Sprintf("conf/machines/%s.json", devfile)

	mc.FlagSet.StringVar(&mc.DevFlag, "dev", devfile, "JSON file containing device metadata")
	mc.FlagSet.StringVar(&mc.FioFlag, "fio", "conf/default", "directory containing fio config templates")

	mc.ParseArgs()

	// load device data from json
	devs := LoadDevicesFile(mustAbs(mc.DevFlag))

	// load the fio config templates into memory
	templates := LoadFioConfDir(mustAbs(mc.FioFlag))

	// use an absolute directory for pathFlag
	outDir := mustAbs(mc.PathFlag)

	// build up a test suite of devs x templates
	suite := NewSuite(mc.IdFlag)
	suite.Populate(devs, templates)
	suite.WriteAll(outDir)
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

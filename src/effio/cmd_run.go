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

	var devFlag, fioFlag string
	var dryrunFlag, rerunFlag bool
	cmd.DefaultFlags()
	cmd.FlagSet.StringVar(&devFlag, "dev", devfile, "JSON file containing device metadata")
	cmd.FlagSet.StringVar(&fioFlag, "fio", "conf/fio/default", "directory containing fio config templates")
	cmd.FlagSet.BoolVar(&dryrunFlag, "dryrun", false, "only generate metadata, without running fio")
	cmd.FlagSet.BoolVar(&rerunFlag, "rerun", false, "only rerun fio benchmarks with missing or empty output.json")
	cmd.ParseArgs()

	if cmd.PathFlag == "" {
		cmd.PathFlag = "./suites/"
	}

	if cmd.NameFlag == "" {
		cmd.FlagSet.Usage()
	}

	// load device data from json
	devs := LoadDevicesFile(mustAbs(devFlag))

	// load the fio config templates into memory
	templates := LoadFioConfDir(mustAbs(fioFlag))

	// use absolute paths for output
	outPath := mustAbs(cmd.PathFlag)

	// build up a test suite of devs x templates
	suite := NewSuite(cmd.NameFlag, outPath)

	// generate all the benchmark permutations
	suite.Populate(devs, templates)

	// filter commands by -incl / -excl if either was specified
	if cmd.InclFlag != "" || cmd.ExclFlag != "" {
		suite.FioCommands = cmd.FilterFioCommands(suite.FioCommands)
	}

	if dryrunFlag {
		fmt.Printf("TODO: print benchmark configs instead of running/writing them.\n")
	} else {
		// write benchmark metadata out under PathFlag
		suite.WriteAll()

		// execute fio commands
		suite.Run(rerunFlag)
	}
}

// FilterFioCommands() filters an FioCommands list by matching fcmd.name
// against -incl / -excl regular expressions and returns an FioCommands
// used by cmd_run.go and cmd_summarize.go
func (cmd *Cmd) FilterFioCommands(in FioCommands) (out FioCommands) {
	out = make(FioCommands, 0)

	for _, fcmd := range in {
		// when no -incl is specified, all tests are included by default
		keep := true
		if cmd.InclFlag != "" {
			// but when one is specified, -incl becomes a whitelisting RE
			keep = false
			if cmd.InclRE.MatchString(fcmd.Name) {
				keep = true
			}
		}

		// blacklist RE always works the same and always comes after -incl
		if cmd.ExclFlag != "" && cmd.ExclRE.MatchString(fcmd.Name) {
			keep = false
		}

		if keep {
			out = append(out, fcmd)
		}
	}

	return out
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

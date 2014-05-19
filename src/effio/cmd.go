package effio

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

type Cmd struct {
	Process string
	Command string
	Args    []string
	FlagSet *flag.FlagSet
}

// NewCmd returns a new command struct with arguments broken out.
// The first argument is always considered a subcommand but is not
// parsed by this package. It is meant to be used with the flag package.
// Usage: cmd := effio.NewCmd(os.Args)
func NewCmd(args []string) (cmd Cmd) {
	if len(args) < 2 {
		cmd.Usage("subcommand required: make|run|inventory|mountall\n")
	}

	cmd.Process = args[0]
	cmd.Command = args[1]

	if len(args) > 2 {
		cmd.Args = args[2:]
	} else {
		cmd.Args = []string{}
	}

	cmd.FlagSet = flag.NewFlagSet(cmd.Process, flag.ExitOnError)
	cmd.FlagSet.Usage = func () {
	    fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	    cmd.FlagSet.PrintDefaults()
		os.Exit(2)
	}

	return cmd
}

func (cmd *Cmd) Run() {
	switch cmd.Command {
	case "make":
		cmd.MakeSuite()
	case "run":
		cmd.RunSuite()
	case "inventory":
		cmd.Inventory()
	case "mountall":
		cmd.Mountall()
	case "help", "-h", "-help", "--help":
		cmd.Usage()
	default:
		cmd.Usage(fmt.Sprintf("Invalid subcommand '%s'.\n", cmd.Command))
	}
}

// TODO: fill in usage when things settle down
func (cmd *Cmd) Usage(more ...string) {
	fmt.Fprintf(os.Stderr, strings.Join(more, ""))
	fmt.Fprintf(os.Stderr, "Usage: %s <command> <args>\n", os.Args[0])
	os.Exit(2)
}

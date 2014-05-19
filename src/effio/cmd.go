package effio

import (
	"flag"
	"fmt"
	"log"
	"os"
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
		log.Fatalf("subcommand required: make|run|inventory|mountall\n%s\n", cmd.Usage())
	}

	cmd.Process = args[0]
	cmd.Command = args[1]

	if len(args) > 2 {
		cmd.Args = args[2:]
	} else {
		cmd.Args = []string{}
	}

	cmd.FlagSet = flag.NewFlagSet(cmd.Process, flag.ExitOnError)

	return cmd
}

func (cmd *Cmd) MinArgs(required int) {
	if len(cmd.Args) < required {
		log.Fatalf("Not enough arguments '%s %s' (%d required)", cmd.Process, cmd.Command, required)
	}
}

func (cmd *Cmd) Run() {
	switch cmd.Command {
	case "make":
		cmd.MinArgs(3)
		cmd.MakeSuite()
	case "run":
		cmd.MinArgs(1)
		cmd.RunSuite()
		cmd.RunSuite()
	case "inventory":
		cmd.MinArgs(0)
		cmd.Inventory()
	case "mountall":
		cmd.MinArgs(0)
		cmd.Mountall()
	case "help", "-h", "-help", "--help":
		fmt.Fprintln(os.Stderr, cmd.Usage())
		os.Exit(2)
	default:
		log.Fatalf("Invalid subcommand '%s'.\n%s\n", cmd.Command, cmd.Usage())
	}
}

// TODO: fill in usage when things settle down
func (cmd *Cmd) Usage() string {
	return fmt.Sprintf("Usage: %s <command> <args>", os.Args[0])
}

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
func NewCmd(args []string) Cmd {
	if len(args) < 3 {
		log.Fatalf("Not enough arguments! (must be at least 2)")
	}

	fs := flag.NewFlagSet(args[0], flag.ExitOnError)

	return Cmd{args[0], args[1], args[2:], fs}
}

func (cmd *Cmd) Usage() string {
	txt := `Usage: %s <command> <args>
	make --device-json <file> --fio-configs <dir> --output <dir>
`
	return fmt.Sprintf(txt, os.Args[0])
}

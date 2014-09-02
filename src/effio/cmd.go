package effio

import (
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
)

type Cmd struct {
	Process  string         // argv[0]
	Command  string         // the subcommand requested, e.g. 'inventory', 'run'
	Args     []string       // args after extracting the subcommand
	InclRE   *regexp.Regexp // compiled regular expression for list filtering
	ExclRE   *regexp.Regexp // compiled regular expression for list filtering
	NameFlag string         // -name
	PathFlag string         // -path
	InclFlag string         // -incl
	ExclFlag string         // -excl
	FlagSet  *flag.FlagSet  // stdlib flag set
}

// NewCmd returns a new command struct with arguments broken out.
// The first argument is always considered a subcommand but is not
// parsed by this package. It is meant to be used with the flag package.
// Usage: cmd := effio.NewCmd(os.Args)
func NewCmd(args []string) (cmd Cmd) {
	if len(args) < 2 {
		cmd.Usage("subcommand required: run|inventory\n")
	}

	cmd.Process = args[0]
	cmd.Command = args[1]

	if len(args) > 2 {
		cmd.Args = args[2:]
	} else {
		cmd.Args = []string{}
	}

	cmd.FlagSet = flag.NewFlagSet(cmd.Process, flag.ExitOnError)
	cmd.FlagSet.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		cmd.FlagSet.PrintDefaults()
		os.Exit(2)
	}

	return cmd
}

func (cmd *Cmd) Run() {
	switch cmd.Command {
	case "run":
		cmd.RunSuite()
	case "inventory":
		cmd.Inventory()
	case "summarize":
		cmd.SummarizeCSV()
	case "help", "-h", "-help", "--help":
		cmd.Usage()
	default:
		cmd.Usage(fmt.Sprintf("Invalid subcommand '%s'.\n", cmd.Command))
	}
}

func (cmd *Cmd) DefaultFlags() {
	cmd.FlagSet.StringVar(&cmd.NameFlag, "name", "", "name of the benchmark")
	cmd.FlagSet.StringVar(&cmd.PathFlag, "path", "", "working directory")

	// -excl is processed after -incl so you can -incl and then pare it down with -excl
	cmd.FlagSet.StringVar(&cmd.InclFlag, "incl", "", "regex matching tests to include in graph")
	cmd.FlagSet.StringVar(&cmd.ExclFlag, "excl", "", "regex matching tests to exclude from graph")
}

func (cmd *Cmd) ParseArgs() {
	var err error
	cmd.FlagSet.Parse(cmd.Args)

	// whitelist
	if cmd.InclFlag != "" {
		cmd.InclRE, err = regexp.Compile(cmd.InclFlag)
		if err != nil {
			log.Fatalf("-incl '%s': regex could not be compiled: %s\n", cmd.InclFlag, err)
		}
	}

	// blacklist, applied after the whitelist
	if cmd.ExclFlag != "" {
		cmd.ExclRE, err = regexp.Compile(cmd.ExclFlag)
		if err != nil {
			log.Fatalf("-excl '%s': regex could not be compiled: %s\n", cmd.ExclFlag, err)
		}
	}
}

// TODO: fill in usage when things settle down
func (cmd *Cmd) Usage(more ...string) {
	fmt.Fprintf(os.Stderr, strings.Join(more, ""))
	fmt.Fprintf(os.Stderr, "Usage: %s <command> <args>\n", os.Args[0])
	os.Exit(2)
}

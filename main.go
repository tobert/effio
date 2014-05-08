package main

import (
	"./src/effio"
	"log"
	"os"
)

/* effio - tools for analyzing fio output
 *
 * Building:
 * go get code.google.com/p/plotinum/plot
 * go build
 *
 * Possible CLI designs:
 *
 *  effio make --devices devices.json --fio-dir fio_configs --out-dir /tmp/test/
 */

func main() {
	// all subcommands require at least one argument, keep it simple for now
	if len(os.Args) < 3 {
		log.Fatalf("Not enough arguments.\n")
	}

	cmd := effio.NewCmd(os.Args)

	switch cmd.Command {
	case "make":
		cmd.Make()
	default:
		log.Fatalf("Invalid subcommand '%s'.\n%s\n", cmd.Command, cmd.Usage())
	}
}

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
 *  effio make_suite -dev devices.json -fio fio_configs/ -out /tmp/test/
 */

func main() {
	cmd := effio.NewCmd(os.Args)

	switch cmd.Command {
	case "make_suite":
		cmd.MakeSuite()
	default:
		log.Fatalf("Invalid subcommand '%s'.\n%s\n", cmd.Command, cmd.Usage())
	}
}

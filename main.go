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
 *  effio make -suite /tmp/test/ -dev devices.json -fio fio_configs/
 */

func main() {
	cmd := effio.NewCmd(os.Args)

	switch cmd.Command {
	case "make":
		cmd.MakeSuite()
	case "run":
		cmd.RunSuite()
	case "inventory":
		cmd.Inventory()
	case "mountall":
		cmd.Mountall()
	default:
		log.Fatalf("Invalid subcommand '%s'.\n%s\n", cmd.Command, cmd.Usage())
	}
}

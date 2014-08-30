package main

import (
	"./src/effio"
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
	cmd.RunCommand()
}

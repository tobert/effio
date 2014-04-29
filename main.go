package main

/* effio - tools for analyzing fio output
 *
 * Building:
 * go get code.google.com/p/plotinum/plot
 * go build
 *
 * Possible CLI designs:
 *
 *  effio json -type line -metric -in <file.json>,<file.json> -md <file> -out <file>
 *  effio latency -in <file.csv>,<file.csv>,... -md <file> -out <file>
 *
 * -md specifies a file of metadata that can be used to create legends & titles
 * {
 *   "devices": {
 *     "/dev/disk/by-id/ata-Samsung_SSD_840_PRO_Series_S1ATNEAD541857W": {
 *       "name": "Samsung 840 Pro SSD",
 *       "color": "blue"
 *   }
 * }
 *
 * License: Apache 2.0
 */

func init() {
}

func main() {
}


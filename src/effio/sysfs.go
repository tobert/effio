package effio

import (
	"io/ioutil"
	"log"
	"path"
	"strconv"
	"strings"
)

func GetSysBlockString(device string, fpath string) string {
	sbpath := path.Join("/sys/block", device, fpath)
	data, err := ioutil.ReadFile(sbpath)
	if err != nil {
		log.Fatal(err)
	}

	return strings.TrimRight(string(data), " \t\r\n")
}

func GetSysBlockInt(device string, fpath string) int64 {
	str := GetSysBlockString(device, fpath)
	out, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		log.Fatal(err)
	}
	return out
}

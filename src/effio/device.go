package effio

// the JSON format is documented in README.md

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

type Device struct {
	Name       string
	Device     string
	Mountpoint string
	Filesystem string
	Brand      string
	Series     string
	Capacity   int64
	Rotational bool
	Transport  string
	HBA        string
	Media      string
	BlockSize  int
}

type Devices []Device

func LoadDevicesFile(fname string) (devs Devices) {
	mdbuf, err := ioutil.ReadFile(fname)
	if err != nil {
		log.Fatalf("Could not read '%s': %s\n", fname, err)
	}
	err = json.Unmarshal(mdbuf, &devs)
	if err != nil {
		log.Fatalf("Could not parse JSON: %s\n", err)
	}

	return devs
}

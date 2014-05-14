package effio

// the JSON format is documented in README.md

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

type Device struct {
	Name       string `json:"name"`
	Device     string `json:"device"`
	Mountpoint string `json:"mountpoint"`
	Filesystem string `json:"filesystem"`
	Brand      string `json:"brand"`
	Series     string `json:"series"`
	Capacity   int64  `json:"capacity"`
	Rotational bool   `json:"rotational"`
	Transport  string `json:"transport"`
	HBA        string `json:"hba"`
	Media      string `json:"media"`
	BlockSize  int    `json:"blocksize"`
	RPM        int    `json:"rpm"`
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

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

// implement the sort interface
func (devs Devices) Len() int {
	return len(devs)
}

func (devs Devices) Swap(i, j int) {
	devs[i], devs[j] = devs[j], devs[i]
}

// sort by mountpoint, which usually ends with the driveletter
// in my setups (for now) ...
func (devs Devices) Less(i, j int) bool {
	return devs[i].Mountpoint < devs[j].Mountpoint
}

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

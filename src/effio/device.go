package effio

// the JSON format is documented in README.md

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path"
	"syscall"
)

type Device struct {
	Name       string `json:"name"`
	Notes      string `json:"notes"`
	Ignore     bool   `json:"ignore"`
	Device     string `json:"device"`
	Mountpoint string `json:"mountpoint"`
	Filesystem string `json:"filesystem"`
	Brand      string `json:"brand"`
	Series     string `json:"series"`
	Datasheet  string `json:"datasheet"`
	Capacity   int64  `json:"capacity"`
	Rotational bool   `json:"rotational"`
	Transport  string `json:"transport"`
	HBA        string `json:"hba"`
	Media      string `json:"media"`
	Blocksize  int    `json:"blocksize"`
	RPM        int    `json:"rpm"`
}

type Devices []Device

func (d *Device) IsMounted() (bool, error) {
	dfi, err := os.Stat(d.Mountpoint)
	if err != nil {
		return false, err
	}

	pfi, err := os.Stat(path.Join(d.Mountpoint, ".."))
	if err != nil {
		return false, err
	}

	// get the underlying OS stat structure (breaking !unix portability)
	dst := dfi.Sys().(*syscall.Stat_t)
	pst := pfi.Sys().(*syscall.Stat_t)

	if dst.Dev == pst.Dev {
		return false, errors.New("mountpoint has same device number as parent directory")
	} else {
		return true, nil
	}
}
func (d *Device) Mount() error {
return nil
}
func (d *Device) Umount() error {
return nil
}

/*
func (d *Device) Mount() error {
	if d.Device == "" {
		return errors.New("'device' must be defined in device json for mounting support")
	}
	if d.Mountpoint == "" {
		return errors.New("'mountpoint' must be defined in device json for mounting support")
	}
	if d.Filesystem == "" {
		return errors.New("'filesystem' must be defined in device json for mounting support")
	}

	if err := os.MkdirAll(d.Mountpoint, 0755); err != nil {
		return err
	}

	var flags uintptr
	flags = syscall.MS_NOATIME | syscall.MS_NODIRATIME
	data := ""

	return syscall.Mount(d.Device, d.Mountpoint, d.Filesystem, uintptr(flags), data)
}

func (d *Device) Umount() error {
	return syscall.Unmount(d.Mountpoint, 0)
}
*/

func (d *Device) ToJson() string {
	js, err := json.MarshalIndent(d, "  ", "  ")
	if err != nil {
		log.Fatalf("Failed to encode device as JSON: %s\n", err)
	}
	return string(js)
}

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

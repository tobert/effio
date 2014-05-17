package effio

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func (cmd *Cmd) Inventory() {
	devs := InventoryDevs()
	sort.Sort(devs)
	js, err := json.MarshalIndent(devs, "", "  ")
	if err != nil {
		log.Fatalf("Failed to encode inventory JSON: %s\n", err)
	}

	fmt.Println(string(js))
}

// in my tests I use whole devices with a single GPT partition and the
// ext4 filesystem (for now). This finds all the devices and grabs most
// of the info needed for the device JSON file and dumps it to stdout
// so it can be put in a file and edited to taste.
func InventoryDevs() (devs Devices) {
	visitor := func(dpath string, f os.FileInfo, err error) error {
		if err != nil {
			log.Fatalf("Encountered an error while inventorying devices '%s': %s", dpath, err)
		}

		fi, err := os.Stat(dpath)
		if err != nil {
			log.Fatal(err)
		}

		// ignore anything that's not a device, os.Stat seems to follow the
		// link and set mode device which is perfect for this case
		if fi.Mode()&os.ModeDevice == 0 {
			return nil
		}

		device := path.Base(dpath)
		reldev, err := os.Readlink(dpath)
		if err != nil {
			log.Fatal(err)
		}
		letter := path.Base(reldev)

		// only consider devices with a partition table and only
		// consider partition 1
		if !strings.HasSuffix(device, "-part1") {
			return nil
		}

		bdev := strings.TrimRight(letter, "1234567890")
		model := GetSysBlockString(bdev, "device/model")
		bsize := GetSysBlockInt(bdev, "queue/hw_sector_size")
		size := GetSysBlockInt(bdev, "size") * bsize
		rotational := GetSysBlockInt(bdev, "queue/rotational")
		brand := GuessBrand(model)
		// lower case, replace spaces and dashes with underscore
		name := strings.Replace(strings.Replace(strings.ToLower(model), " ", "_", -1), "-", "_", -1)

		d := Device{
			Name:       name,
			Device:     dpath,
			Mountpoint: path.Join("/mnt/effio", name),
			Filesystem: "ext4",
			Brand:      brand,
			Series:     model,
			Capacity:   size,
			Rotational: (rotational == 1),
			Transport:  "", // can be detected but it's a lot of work
			HBA:        "", // ditto
			Media:      "", // no way to detect
			Blocksize:  int(bsize),
			RPM:        0, // no way to detect?
		}

		devs = append(devs, d)

		return nil
	}

	err := filepath.Walk("/dev/disk/by-id", visitor)
	if err != nil {
		log.Fatalf("Could not inventory devices in /dev/disk/by-id: %s", err)
	}

	return devs
}

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

func GuessBrand(model string) string {
	if strings.HasPrefix(model, "Samsung") {
		return "Samsung"
	} else if strings.HasPrefix(model, "ST") {
		return "Seagate"
	} else if strings.HasPrefix(model, "WD") {
		return "Western Digital"
	} else if strings.HasPrefix(model, "MRD") {
		return "I/O Switch"
	} else if strings.HasPrefix(model, "SSD") {
		return "PNY"
	} else {
		return "Generic"
	}
}

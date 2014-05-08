package effio

/* The JSON device file is an array of devices and some relevant data. e.g.
 *
 * [
 *   {
 *     "name":       "samsung_840_pro_256",
 *     "device":     "/dev/disk/by-id/ata-Samsung_SSD_840_PRO_Series_S1ATNEAD541857W",
 *     "mountpoint": "/mnt/sda",
 *     "filesystem": "ext4",
 *     "brand":      "Samsung",
 *     "series":     "840 PRO",
 *     "capacity":   256060514304,
 *     "rotational": false,
 *     "transport":  "SATA",
 *     "hba":        "AHCI",
 *     "media":      "MLC",
 *     "blocksize":  512
 *   }
 * ]
 *
 * Descriptions:
 *   name:       manually assigned, will be used in file names!
 *   device:     always use the /dev/disk/by-id/ path
 *   mountpoint: location where the filesystem is mounted
 *   filesystem: ext4, xfs, zfs, btrfs, ntfs-3g
 *   brand:      Samsung, Fusion IO, I/O Switch Tech, Seagate, Western Digital, etc.
 *   series:     "840 PRO",
 *   capacity:   `blockdev --getsize64 /dev/sda`
 *   rotational: false for SSD, true for HDD, true if device contains any HDD
 *   transport:  SATA, SAS, PCIe, MDRAID, iSCSI, virtio
 *   hba:        ioMemory, AHCI, SAS3004, USB3, mixed (for MDRAID)
 *   media:      MLC, Iron (for HDDs), TLC, SLC, Hybrid (SSHD)
 *   blocksize:  `blockdev --getpbsz /dev/sda`
 */

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

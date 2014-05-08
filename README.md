effio
=====

[COMING SOON](http://tobert.github.io/post/2014-04-29-a-quick-prototype.html)

Tools for running suites of tests with fio, capturing the output, then generating reports.

Usage
-----

```
./effio make_suite -id 2014-05-07 \
           -dev ./conf/machines/brak.tobert.org.json \
           -fio ./conf/fio_disk_latency \
           -out ./out
```

Device JSON Format
------------------

The JSON device file is an array of devices and some relevant data. e.g.

Check out the
[example file](https://github.com/tobert/effio/blob/master/conf/machines/brak.tobert.org.json)

```json
[
   {
     "name":       "samsung_840_pro_256",
     "device":     "/dev/disk/by-id/ata-Samsung_SSD_840_PRO_Series_S1ATNEAD541857W",
     "mountpoint": "/mnt/sda",
     "filesystem": "ext4",
     "brand":      "Samsung",
     "series":     "840 PRO",
     "capacity":   256060514304,
     "rotational": false,
     "transport":  "SATA",
     "hba":        "AHCI",
     "media":      "MLC",
     "blocksize":  512
   }
]
```

Field      | Description
-----------|-------------
name       | manually assigned, will be used in file names!
device     | always use the /dev/disk/by-id/ path
mountpoint | location where the filesystem is mounted
filesystem | ext4, xfs, zfs, btrfs, ntfs-3g
brand      | Samsung, Fusion IO, I/O Switch Tech, Seagate, Western Digital, etc.
series     | "840 PRO",
capacity   | `blockdev --getsize64 /dev/sda`
rotational | false for SSD, true for HDD, true if device contains any HDD
transport  | SATA, SAS, PCIe, MDRAID, iSCSI, virtio
hba        | ioMemory, AHCI, SAS3004, USB3, mixed (for MDRAID)
media      | MLC, Iron (for HDDs), TLC, SLC, Hybrid (SSHD)
blocksize  | `blockdev --getpbsz /dev/sda`


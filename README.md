effio
=====

[COMING SOON](http://tobert.github.io/post/2014-04-29-a-quick-prototype.html)

Tools for running suites of tests with fio, capturing the output, then generating reports.

This tool establishes and automates a few conventions for managing tests with the goal
of making those tests manageable and repeatable.

Usage
-----

```
./effio make_suite -id 2014-05-07 \
           -dev ./conf/machines/brak.tobert.org.json \
           -fio ./conf/fio_disk_latency \
           -out ./out
```

### Subcommands

##### `effio make_suite -id <string> -dev <file.json> -fio <dir> -out <dir>`

All arguments are required. This command takes a set of fio configuration
and a JSON file defining the devices to be tested and produces a new
tree of files containing all of the data required to run the suite. The
tests are a cartesian product of all fio configs x all devices in the json.

Fio configuration files are run through text/template with data from the device
and other derived strings available.

* `-id string` a unique id for the generated suite, should be a suitable directory name
* `-dev file.json` a file populated with device information, documented below
* `-fio dir` a directory containing fio configuration files
* `-out dir` the suite will be written under this dir with id as the first entry

The directory structure will look something like this, given -id 'foo', the device
json below, and one fio test in the -fio dir, dir/rand_512b_write_iops.fio.

```
dir/
    foo/
        suite.json    # a dump of all information related to the suite
        rand_512b_write_iops-samsung_840_pro_256/
          config.fio  # the fio configuration file
          test.json   # a dump of all data used to generate this test
          output.json # json output from fio --output-format=json
          run.sh      # the exact command used to run fio
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


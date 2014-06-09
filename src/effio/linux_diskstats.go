package effio

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"syscall"
	"time"
)

// sample:
//     0       1 2    3       4    5        6      7        8      9         10      11 12      13
// "   8       1 sda1 3774108 2444 30194552 637303 11746691 283841 529382982 90114806 0 1657750 90747516"
//   %4d     %7d %s   %lu     %lu  %lu      %u     %lu      %lu    %lu       %u      %u %u      %u\n
// ^ from linux/block/genhd.c ~ line 1139
type Diskstat struct {
	Major         uint          //  0: major dev no
	Minor         uint          //  1: minor dev no
	Name          string        //  2: device name
	ReadComplete  uint64        //  3: reads completed
	ReadMerged    uint64        //  4: writes merged
	ReadSectors   uint64        //  5: sectors read
	ReadMs        uint          //  6: ms spent reading
	WriteComplete uint64        //  7: reads completed
	WriteMerged   uint64        //  8: writes merged
	WriteSectors  uint64        //  9: sectors read
	WriteMs       uint          // 10: ms spent writing
	IOPending     uint          // 11: number of IOs currently in progress
	IOMs          uint          // 12: jiffies_to_msecs(part_stat_read(hd, io_ticks))
	IOQueueMs     uint          // 13: jiffies_to_msecs(part_stat_read(hd, time_in_queue))
	Time          time.Time     // time that /proc/diskstats was read
	Duration      time.Duration // only used in deltas
}

type Diskstats []Diskstat

// start a goroutine that will get stats from /proc/diskstats for the given
// major/minor and write to the named file in CSV format every second
// until the returned channel is closed, at which time the goroutine
// will close the file then exit
// finish := CollectDiskstats("/tmp/test.dat", 8, 49)
// do stuff ..
// close(finish)
func CollectDiskstats(fname string, d Device) chan struct{} {
	finish := make(chan struct{})
	major, minor := d.devNums()

	go func() {
		fd, err := os.OpenFile(fname, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			log.Fatalf("Could not open file '%s' for writing diskstats: %s", fname, err)
		}
		defer fd.Close()

		for {
			select {
			case <-time.After(time.Second):
				stats := ReadDiskstats()
				for _, st := range stats {
					if st.Major == major && st.Minor == minor {
						//                t  0  1  2  3  4  5  6  7  8  9 10 11 12 13
						fmt.Fprintf(fd, "%d,%d,%d,%s,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d\n",
							st.Time.UnixNano(), st.Major, st.Minor, st.Name, // t,0,1,2
							st.ReadComplete, st.ReadMerged, st.ReadSectors, st.ReadMs, // 3,4,5,6
							st.WriteComplete, st.WriteMerged, st.WriteSectors, st.WriteMs, // 7,8,9,10
							st.IOPending, st.IOMs, st.IOQueueMs) // 11,12,13
					}
				}
			case <-finish:
				return
			}
		}
	}()

	return finish
}

func ReadDiskstats() (out Diskstats) {
	data, err := ioutil.ReadFile("/proc/diskstats")
	if err != nil {
		log.Fatalf("Could not read /proc/diskstats: %s\n", err)
	}

	timestamp := time.Now()

	// slice off the last byte, a newline, to prevent a phantom row
	rows := bytes.Split(data[0:len(data)-1], []byte{byte('\n')})

	// bytes.Split doesn't handle variable whitespace between fields
	fields := make([]string, 14)
	field := make([]byte, 32)
	var f, i int
	for _, row := range rows {
		f = 0
		i = 0
		for _, b := range row[0 : len(row)-1] {
			if b != byte(' ') {
				field[i] = b
				i++
			} else if i > 0 {
				fields[f] = string(field[0:i])
				f++
				i = 0
			}
		}

		st := Diskstat{
			fldtoint(fields, 0),
			fldtoint(fields, 1),
			fields[2],
			fldtoint64(fields, 3),
			fldtoint64(fields, 4),
			fldtoint64(fields, 5),
			fldtoint(fields, 6),
			fldtoint64(fields, 7),
			fldtoint64(fields, 8),
			fldtoint64(fields, 9),
			fldtoint(fields, 10),
			fldtoint(fields, 11),
			fldtoint(fields, 12),
			fldtoint(fields, 13),
			timestamp,
			0,
		}

		out = append(out, st)
	}

	return out
}

func (from *Diskstat) Delta(to Diskstat) Diskstat {
	if from.Major != to.Major || from.Minor != to.Minor {
		log.Fatal("Comparing different devices doesn't make sense. %s / %s\n", from.Name, to.Name)
	}

	return Diskstat{
		to.Major,
		to.Minor,
		to.Name,
		to.ReadComplete - from.ReadComplete,
		to.ReadMerged - from.ReadMerged,
		to.ReadSectors - from.ReadSectors,
		to.ReadMs - from.ReadMs,
		to.WriteComplete - from.WriteComplete,
		to.WriteMerged - from.WriteMerged,
		to.WriteSectors - from.WriteSectors,
		to.WriteMs - from.WriteMs,
		to.IOPending - from.IOPending,
		to.IOMs - from.IOMs,
		to.IOQueueMs - from.IOQueueMs,
		to.Time,
		to.Time.Sub(from.Time),
	}
}

func fldtoint(fields []string, idx int) uint {
	return uint(fldtoint64(fields, idx))
}

func fldtoint64(fields []string, idx int) uint64 {
	if len(fields[idx]) == 0 {
		return 0
	}

	out, err := strconv.ParseUint(fields[idx], 10, 64)
	if err != nil {
		log.Fatalf("Failed to convert field %d, value '%s', device '%s' to int: %s\n",
			idx, fields[idx], fields[2], err)
	}

	return out
}

// major/minor decoding from /usr/include/sys/sysmacros.h on linux
func (d *Device) devNums() (major, minor uint) {
	fi, err := os.Stat(d.Device)
	if err != nil {
		log.Fatalf("Could not stat device file: %s\n", err)
	}
	native := fi.Sys().(*syscall.Stat_t)
	// http://stackoverflow.com/questions/4309882/device-number-in-stat-command-output
	major = uint(native.Rdev) >> 8 & 0xff
	minor = uint(native.Rdev) & 0xff

	return
}

package data

import (
	"clearview/agent/utils"
	"clearview/common"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"syscall"
)

func GetDataDisks(data *common.Data) error {

	data.Disk = &common.DataDisk{}

	// Get swap device names
	swapDeviceMap := getSwapDevicesAsMap()
	diskDeviceMap := make(map[string]*common.DataDiskItem)

	// Parse mtab for mounted devices
	mtabData, err := ioutil.ReadFile("/etc/mtab")
	if err != nil {
		return err
	}

	for _, line := range strings.Split(string(mtabData), "\n") {
		if strings.HasPrefix(line, "/") {
			m := strings.Fields(line)
			if len(m) >= 2 {
				devName := m[0]
				devPath := m[1]

				/*
					TODO

							if ( $device =~ m|^/dev/mapper| ) {
							my $linkpath = readlink($device);
							if ($linkpath) {
								$device = abs_path("/dev/mapper/$linkpath");
							}
							else {
								my $rdev=(stat($device))[6];
								my $minor_m = ($rdev & 037774000377) >> 0000000;
								$device = "/dev/dm-$minor_m";
							}
						}
				*/

				if devName == "/dev/root" {
					psCmdLine, err := utils.ReadProcFSFile("cmdline")
					if err == nil {
						cmdLine := psCmdLine.GetAsString()
						rootBegin := strings.Index(cmdLine, "root=")
						if rootBegin >= 0 {
							rootEnd := strings.Index(cmdLine[rootBegin:], " ")
							if rootEnd < 0 {
								rootEnd = len(cmdLine) - rootBegin
							}
							rootEnd += rootBegin
							rootBegin += 5
							rootStr := cmdLine[rootBegin:rootEnd]
							devName = rootStr
						}
					}
				}

				if strings.HasPrefix(devName, "UUID=") {
					devLink, err := os.Readlink("/dev/disk/by-uuid/" + devName[5:])
					if err == nil {
						devName = path.Join("/dev/disk/by-uuid", devLink)
					}
				}

				if strings.HasPrefix(devName, "/dev/") {
					devName = devName[5:]
				}

				disk := &common.DataDiskItem{
					Name:      devName,
					Path:      devPath,
					IsMounted: true,
				}

				var statFs syscall.Statfs_t
				if syscall.Statfs(devPath, &statFs) == nil {
					disk.BytesFree = uint64(statFs.Bsize) * statFs.Bfree
					disk.BytesTotal = uint64(statFs.Bsize) * statFs.Blocks
					disk.INodeFree = statFs.Ffree
					disk.INodeTotal = statFs.Files
				}

				data.Disk.DiskList = append(data.Disk.DiskList, disk)
				diskDeviceMap[disk.Name] = disk
			}
		}
	}

	for _, swap := range swapDeviceMap {
		if _, ok := diskDeviceMap[swap.DevName]; !ok {
			disk := &common.DataDiskItem{
				Name:       swap.DevName,
				Path:       "swap-" + swap.DevName,
				IsMounted:  true,
				IsSwap:     true,
				BytesTotal: swap.Size,
				BytesFree:  swap.Size - swap.Used,
			}
			data.Disk.DiskList = append(data.Disk.DiskList, disk)
			diskDeviceMap[disk.Name] = disk
		}
	}

	psDiskStats, err := utils.ReadProcFSFile("diskstats")
	if err != nil {
		return err
	}

	for _, l := range psDiskStats.GetAsLines() {
		s := strings.Fields(l)
		if len(s) >= 10 {

			/*
				TODO

					if (substr($device,0,8) eq '/dev/dm-') {
						# if the filesystem sees it under /dev
						if ( -b $device ) {
							unless (keys(%dev_mapper)) {
								%dev_mapper = map { substr(readlink($_),3) => substr($_,12); } (glob("/dev/mapper/*"));
							}
							if (exists($dev_mapper{substr($device,5)})) {
								$dataref->{INSTANT}->{"Disk.$e_device.label"} = $dev_mapper{substr($device,5)};
							}
						} else {
							unless (keys(%dev_mapper)) {
								%dev_mapper = map {
									my $rdev=(stat($_))[6];
									my $major_m = ($rdev & 03777400) >> 0000010;
									my $minor_m = ($rdev & 037774000377) >> 0000000;
									join('_', $major_m,$minor_m) => substr($_,12);
								} glob ("/dev/mapper/*");
							}
							if (exists($dev_mapper{$major."_".$minor})) {
								$dataref->{INSTANT}->{"Disk.$e_device.label"} = $dev_mapper{$major."_".$minor};
							}
						}
					} elsif ($device =~ m|(/dev/md\d+)(p\d+)?|) {

			*/

			devName := s[2]
			readCount, _ := strconv.ParseUint(s[3], 10, 64)
			readSectors, _ := strconv.ParseUint(s[5], 10, 64)
			writeCount, _ := strconv.ParseUint(s[7], 10, 64)
			writeSectors, _ := strconv.ParseUint(s[9], 10, 64)

			if readCount != 0 || writeCount != 0 {
				sectorSize := getHwSectorSize(devName)

				var disk *common.DataDiskItem

				if d, ok := diskDeviceMap[devName]; ok {
					disk = d
				} else {
					disk = &common.DataDiskItem{
						Name: devName,
						Path: "",
					}
				}

				if _, ok := swapDeviceMap[devName]; ok {
					disk.IsSwap = true
				}

				disk.ReadCountCumulative = readCount
				disk.WriteCountCumulative = writeCount
				disk.ReadBytesCumulative = readSectors * sectorSize
				disk.WriteBytesCumulative = writeSectors * sectorSize

				data.Disk.Combined.ReadCountCumulative += readCount
				data.Disk.Combined.WriteCountCumulative += writeCount
				data.Disk.Combined.ReadBytesCumulative += readSectors * sectorSize
				data.Disk.Combined.WriteBytesCumulative += writeSectors * sectorSize
			}
		}
	}

	return nil
}

type swapDevice struct {
	DevName string
	Size    uint64
	Used    uint64
}

func getSwapDevicesAsMap() map[string]*swapDevice {
	m := make(map[string]*swapDevice)

	psSwaps, err := utils.ReadProcFSFile("swaps")
	if err != nil {
		return m
	}

	for _, l := range psSwaps.GetAsLines() {
		if strings.HasPrefix(l, "/") {
			v := strings.Fields(l)
			if len(v) > 0 && v[1] == "partition" {
				devName := v[0]

				if strings.HasPrefix(devName, "/dev/") {
					devName = devName[5:]
				}

				devItem := &swapDevice{
					DevName: devName,
				}
				devItem.Size, _ = strconv.ParseUint(v[2], 10, 64)
				devItem.Used, _ = strconv.ParseUint(v[3], 10, 64)

				devItem.Size *= 1024
				devItem.Used *= 1024

				m[devName] = devItem

			}
		}
	}

	return m
}

func getHwSectorSize(devName string) uint64 {
	sectorSize := uint64(512)

	if size, err := getHwSectorSizeRaw(devName); err == nil {
		sectorSize = size
	} else {
		devParent := devName
		for {
			l := len(devParent)
			if l > 1 {
				if ch := devParent[l-1]; ch >= '0' && ch <= '9' {
					devParent = devParent[:l-1]
				} else {
					break
				}
			} else {
				break
			}
		}

		if devParent != devName {
			if size, err := getHwSectorSizeRaw(devName); err == nil {
				sectorSize = size
			}
		}
	}

	return sectorSize
}

func getHwSectorSizeRaw(devName string) (uint64, error) {
	data, err := ioutil.ReadFile("/sys/block/" + devName + "/queue/hw_sector_size")
	if err != nil {
		return 0, err
	}

	size, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0, err
	}

	return size, nil
}

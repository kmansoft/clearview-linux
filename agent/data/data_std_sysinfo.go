package data

import (
	"clearview/agent/utils"
	"clearview/common"
	"io/ioutil"
	"strconv"
	"strings"
	"syscall"
)

func GetDataSysInfo(data *common.Data) error {

	data.SysInfo = &common.DataSystemInfo{
		DataSystemMemory: common.DataSystemMemory{
			RealMemorySize: data.Memory.RealFree + data.Memory.RealUsed,
			RealMemoryUsed: data.Memory.RealUsed,
		},
	}

	for _, disk := range data.Disk.DiskList {
		if disk.IsMounted {
			if disk.IsSwap {
				data.SysInfo.SwapMemorySize += disk.BytesTotal
				data.SysInfo.SwapMemoryUsed += disk.BytesTotal - disk.BytesFree
			} else {
				data.SysInfo.DiskTotalSize += disk.BytesTotal
				data.SysInfo.DiskTotalUsed += disk.BytesTotal - disk.BytesFree
			}
		}
	}

	// Swap files
	psSwap, err := utils.ReadProcFSFile("swaps")
	if psSwap != nil && err == nil {
		for _, line := range psSwap.GetAsLines() {
			f := strings.Fields(line)
			if len(f) >= 5 && f[1] == "file" {
				size, _ := strconv.ParseUint(f[2], 10, 32)
				used, _ := strconv.ParseUint(f[3], 10, 32)

				data.SysInfo.SwapMemorySize += 1024 * size
				data.SysInfo.SwapMemoryUsed += 1024 * used
			}
		}
	}

	// Processor name
	psCpuInfo, err := utils.ReadProcFSFile("cpuinfo")
	if err != nil {
		return err
	}
	modelName, _ := psCpuInfo.GetStringValue(`model name\s+:`)
	data.SysInfo.CpuLabel = modelName

	// Kernel name
	var u syscall.Utsname
	err = syscall.Uname(&u)
	if err == nil {
		data.SysInfo.KernelLabel = utils.ConvertCharsToString(u.Sysname) + " " +
			utils.ConvertCharsToString(u.Release)
	}

	// System name
	// Distro name and version
	bytes, err := ioutil.ReadFile("/etc/os-release")
	if err != nil {
		return err
	}

	for _, l := range strings.Split(string(bytes), "\n") {
		i := strings.Index(l, "=")
		if i > 0 {
			key := l[0:i]
			value := l[i+1:]

			j := len(value)
			if j >= 2 && value[0] == '"' && value[j-1] == '"' {
				value = value[1 : j-1]
			}

			if key == "NAME" {
				data.SysInfo.OsDistLabel = value
			} else if key == "VERSION_ID" {
				data.SysInfo.OsVersionLabel = value
			}
		}
	}

	return nil
}

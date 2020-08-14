package data

import (
	"clearview/agent/utils"
	"clearview/common"
)

func GetDataMemory(data *common.Data) error {
	psMemInfo, err := utils.ReadProcFSFile("meminfo")
	if err != nil {
		return err
	}

	realFree, _ := psMemInfo.GetNumberValue("MemFree:")
	realUsed, _ := psMemInfo.GetNumberValue("MemTotal:")
	realUsed -= realFree
	realBuffers, _ := psMemInfo.GetNumberValue("Buffers:")
	realCache, _ := psMemInfo.GetNumberValue("Cached:")

	swapFree, _ := psMemInfo.GetNumberValue("SwapFree:")
	swapUsed, _ := psMemInfo.GetNumberValue("SwapTotal:")
	swapUsed -= swapFree

	data.Memory = &common.DataMemory{
		RealUsed:    realUsed * 1024,
		RealFree:    realFree * 1024,
		RealBuffers: realBuffers * 1024,
		RealCache:   realCache * 1024,
		SwapUsed:    swapUsed * 1024,
		SwapFree:    swapFree * 1024,
	}

	return nil
}

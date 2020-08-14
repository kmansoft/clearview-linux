package data

import (
	"clearview/agent/utils"
	"clearview/common"
	"regexp"
	"strconv"
	"strings"
)

func GetDataCPU(data *common.Data) error {

	psStat, err := utils.ReadProcFSFile("stat")
	if err != nil {
		return err
	}

	expr := regexp.MustCompile(`^cpu(\d+)`)

	cpuList := make([]*common.DataCpuItem, 0)
	cpuCount := uint64(0)
	combined := common.DataCpuItem{}

	for i, l := range psStat.GetAsLines() {
		if i > 0 {
			m := expr.FindStringSubmatch(l)
			if len(m) == 2 {
				if cpuN, err := strconv.ParseUint(m[1], 10, 32); err == nil && cpuN < 64 {
					if cpuCount < cpuN+1 {
						cpuCount = cpuN + 1
					}

					r := strings.Fields(l)
					if len(r) >= 8 {
						user, _ := strconv.ParseUint(r[1], 10, 64)
						nice, _ := strconv.ParseUint(r[2], 10, 64)
						system, _ := strconv.ParseUint(r[3], 10, 64)
						wait, _ := strconv.ParseUint(r[5], 10, 64)

						cpuList = append(cpuList, &common.DataCpuItem{
							N:                cpuN,
							UserCumulative:   user + nice,
							SystemCumulative: system,
							WaitCumulative:   wait,
						})
						combined.UserCumulative += user + nice
						combined.SystemCumulative += system
						combined.WaitCumulative += wait
					}
				}
			}
		}
	}

	loadAvg := 0.0
	psLoadAvg, err := utils.ReadProcFSFile("loadavg")
	if err != nil {
		return err
	}

	line1 := strings.Split(psLoadAvg.GetAsString(), " ")
	if len(line1) > 1 {
		loadAvg, _ = strconv.ParseFloat(line1[0], 64)
	}

	data.Cpu = &common.DataCpu{
		CpuCount: cpuCount,
		CpuList:  cpuList,
		Combined: combined,
		LoadAvg:  loadAvg,
	}

	return nil
}

package data

import (
	"clearview/agent/utils"
	"clearview/common"
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
)

func GetDataNetwork(data *common.Data) error {

	// Interface stats
	psNetDev, err := utils.ReadProcFSFile("net/dev")
	if err != nil {
		return err
	}

	eth := ""
	expr := regexp.MustCompile(`^\s+([a-zA-Z0-9]+):\s+\d+`)

	listInterface := make([]*common.DataNetworkItem, 0)
	combined := common.DataNetworkItem{Name: "combined"}

	for _, l := range psNetDev.GetAsLines() {
		m := expr.FindStringSubmatch(l)
		if len(m) == 2 {
			name := m[1]
			fields := strings.Fields(l)
			rx, _ := strconv.ParseUint(fields[1], 10, 64)
			tx, _ := strconv.ParseUint(fields[9], 10, 64)

			if rx != 0 || tx != 0 {
				listInterface = append(listInterface, &common.DataNetworkItem{
					Name:         name,
					RxCumulative: rx,
					TxCumulative: tx,
				})

				if !strings.HasPrefix(name, "lo") {
					combined.RxCumulative += rx
					combined.TxCumulative += tx

					if eth == "" {
						eth = name
					}
				}
			}
		}
	}

	// Linode uses eth0 but for testing we need to support other names (eno1 on Fedora)
	mac, err := ioutil.ReadFile("/sys/class/net/eth0/address")
	if err != nil && eth != "" {
		mac, err = ioutil.ReadFile(fmt.Sprintf("/sys/class/net/%s/address", eth))
	}

	data.Network = &common.DataNetwork{
		MacAddr:   strings.TrimSpace(string(mac)),
		Combined:  combined,
		Interface: listInterface,
	}

	// IP4 stats
	psNetStat, err := utils.ReadProcFSFile("net/netstat")
	if err != nil {
		return err
	}

	for _, l := range psNetStat.GetAsLines() {
		if strings.HasPrefix(l, "IpExt:") {
			f := strings.Fields(l)
			if len(f) >= 8 {
				data.Network.Ip46.RxIP4Cumulative, err = strconv.ParseUint(f[7], 10, 64)
				data.Network.Ip46.TxIP4Cumulative, err = strconv.ParseUint(f[8], 10, 64)
			}
		}
	}

	// IP6 stats
	psNetSnmp6, err := utils.ReadProcFSFile("net/snmp6")
	if err != nil {
		return err
	}

	for _, l := range psNetSnmp6.GetAsLines() {
		if strings.Index(l, "Octet") >= 0 {
			f := strings.Fields(l)
			if f[0] == "Ip6InOctets" {
				data.Network.Ip46.RxIP6Cumulative, err = strconv.ParseUint(f[1], 10, 64)
			} else if f[0] == "Ip6OutOctets" {
				data.Network.Ip46.RxIP6Cumulative, err = strconv.ParseUint(f[1], 10, 64)
			}
		}
	}

	return nil
}

package data

import (
	"clearview/common"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/user"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

func GetDataProcessesPorts(data *common.Data) error {

	// Processes
	err := newProcessList(data)
	if err != nil {
		return err
	}

	for _, proc := range data.Processes {
		if proc.PID == 2 || proc.PPID == 2 {
			continue
		}
	}

	/*

		TODO

			for my $key ( sort keys %{ $dataref->{LONGTERM} } ) {
				next unless $key =~ /^Processes/;
				if($key =~ /zz_age$/ ){
					delete $dataref->{LONGTERM}->{$key};
					next;
				}
				next if ($key =~ m/^Processes\.apt(?:-get|itude)/);
				my $age_key = $key;
				$age_key =~ s/[^\.]*$/zz_age/;
				delete $dataref->{LONGTERM}->{$key} if ($dataref->{LONGTERM}->{$age_key} < 60);
			}

	*/

	// Ports (network connections)
	networkList, err := newNetworkList()
	if err != nil {
		return err
	}

	active := make(map[string]*common.DataPortsActiveItem)
	listen := make(map[string]*common.DataPortsListenItem)

	myPid := os.Getpid()

	for _, proc := range data.Processes {
		if proc.PID == 2 || proc.PPID == 2 {
			continue
		}

		if int(proc.PID) == myPid {
			continue
		}

		files, err := ioutil.ReadDir(fmt.Sprintf("/proc/%d/fd", proc.PID))
		if err != nil {
			continue
		}

		for _, fd := range files {
			if rl, err := os.Readlink(fmt.Sprintf("/proc/%d/fd/%s", proc.PID, fd.Name())); err == nil {
				if len(rl) > 0 && strings.HasPrefix(rl, "socket:") {
					socket := rl[7:]
					sockl := len(socket)
					if sockl > 2 && socket[0] == '[' && socket[sockl-1] == ']' {
						socket = socket[1 : sockl-1]
						if inode, err := strconv.ParseUint(socket, 10, 64); err == nil {
							if network, ok := networkList.networkByINode[inode]; ok {

								if network.isListening {
									// Listening
									key := fmt.Sprintf("%s.%s.%s.%s.%d",
										proc.Name, proc.User,
										network.t, network.srcAddr.String(), network.srcPort)
									if _, ok := listen[key]; !ok {
										listen[key] = &common.DataPortsListenItem{
											Name: proc.Name, User: proc.User,
											Type:    network.t,
											SrcAddr: network.srcAddr, SrcPort: network.srcPort}
									}
								} else {
									// Active
									key := fmt.Sprintf("%s.%s",
										proc.Name, proc.User)
									if activeItem, ok := active[key]; ok {
										activeItem.Count += 1
									} else {
										active[key] = &common.DataPortsActiveItem{
											Name: proc.Name, User: proc.User,
											Count: 1,
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	activeList := make([]*common.DataPortsActiveItem, 0)
	for _, activeItem := range active {
		activeList = append(activeList, activeItem)
	}

	listenList := make([]*common.DataPortsListenItem, 0)
	for _, listenItem := range listen {
		listenList = append(listenList, listenItem)
	}

	data.Ports = &common.DataPorts{
		Active: activeList,
		Listen: listenList,
	}

	return nil
}

func newProcessList(data *common.Data) error {
	data.Processes = make([]*common.DataProcess, 0)

	err := loadProcessList(data)
	if err != nil {
		return err
	}

	return nil
}

func loadProcessList(data *common.Data) error {
	expr := regexp.MustCompile(`\d+`)

	// Read /proc and save all processes
	files, err := ioutil.ReadDir("/proc")
	if err != nil {
		return err
	}

	processItemMap := make(map[string]*common.DataProcess)

	for _, f := range files {
		n := f.Name()
		if expr.MatchString(n) {
			pid, err := strconv.ParseUint(n, 10, 64)
			if err != nil {
				continue
			}

			item := getProcessItem(data, pid)
			if item != nil {
				prefix := item.Name + "." + item.User

				if existing, ok := processItemMap[prefix]; ok {
					existing.Count += 1

					existing.IOReadCharsCumulative += item.IOReadCharsCumulative
					existing.IOWriteCharsCumulative += item.IOWriteCharsCumulative
					existing.IOReadBytesCumulative += item.IOReadBytesCumulative
					existing.IOWriteBytesCumulative += item.IOWriteBytesCumulative
					existing.RSS += item.RSS
					existing.CPUCumulative += item.CPUCumulative
				} else {
					data.Processes = append(data.Processes, item)
					processItemMap[prefix] = item
				}
			}
		}
	}

	return nil
}

func getProcessItem(data *common.Data, pid uint64) *common.DataProcess {
	prefix := fmt.Sprintf("/proc/%d/", pid)

	if status, err := ioutil.ReadFile(prefix + "status"); err == nil {
		if stat, err := ioutil.ReadFile(prefix + "stat"); err == nil {
			if cmdline, err := ioutil.ReadFile(prefix + "cmdline"); err == nil {
				io, err := ioutil.ReadFile(prefix + "io")
				if err != nil {
					io = nil
				}

				item := &common.DataProcess{
					PID:     pid,
					Command: getNullTerminatedString(cmdline),
					Count:   1,
				}

				// Parse "status"
				for _, l := range strings.Split(string(status), "\n") {
					i := strings.IndexByte(l, ':')
					if i > 0 {
						key := l[:i]
						val := strings.TrimSpace(l[i+1:])

						switch key {
						case "Name":
							item.Name = val
						case "PPid":
							item.PPID, _ = strconv.ParseUint(val, 10, 64)
						case "Uid":
							item.UID, _ = strconv.ParseUint(getField(val, 1), 10, 64)
						case "VmRSS":
							item.RSS, _ = strconv.ParseUint(getField(val, 0), 10, 64)
						}
					}
				}

				item.RSS *= 1024

				if item.UID >= 0 {
					u, err := user.LookupId(strconv.FormatUint(item.UID, 10))
					if err == nil {
						item.User = u.Username
					}
				}

				// Parse "stat"
				f := strings.Fields(string(stat))
				if len(f) > 21 {
					userTime, _ := strconv.ParseUint(f[13], 10, 64)
					systemTime, _ := strconv.ParseUint(f[14], 10, 64)
					start, _ := strconv.ParseUint(f[21], 10, 64)

					item.CPUCumulative = userTime + systemTime
					item.Age = processAge(data, start)
				}

				// Parse "io"
				for _, l := range strings.Split(string(io), "\n") {
					i := strings.IndexByte(l, ':')
					if i > 0 {
						key := l[:i]
						val := strings.TrimSpace(l[i+1:])

						switch key {
						case "rchar":
							item.IOReadCharsCumulative, _ = strconv.ParseUint(val, 10, 64)
						case "wchar":
							item.IOWriteCharsCumulative, _ = strconv.ParseUint(val, 10, 64)
						case "read_bytes":
							item.IOReadBytesCumulative, _ = strconv.ParseUint(val, 10, 64)
						case "write_bytes":
							item.IOWriteBytesCumulative, _ = strconv.ParseUint(val, 10, 64)
						}
					}
				}

				return item
			}
		}
	}

	return nil
}

func processAge(data *common.Data, start uint64) uint64 {
	if data.UptimeTicks > 0 && data.TicksPerSecond > 0 {
		current := data.UptimeTicks * data.TicksPerSecond
		return uint64((current - start) / data.TicksPerSecond)
	}

	return 0
}

type DataNetworkList struct {
	needFlip       bool
	networkByINode map[uint64]DataNetwork
}

type DataNetwork struct {
	t                string
	srcAddr, dstAddr net.IP
	srcPort, dstPort uint16
	isListening      bool
}

func newNetworkList() (*DataNetworkList, error) {
	list := &DataNetworkList{}

	err := list.loadNetworkCache()
	if err != nil {
		return nil, err
	}

	return list, nil
}

func (list *DataNetworkList) loadNetworkCache() error {
	arch := runtime.GOARCH

	list.needFlip = arch == "amd64" || arch == "x86"
	list.networkByINode = make(map[uint64]DataNetwork)

	for _, t := range []string{"tcp", "tcp6", "udp", "udp6"} {
		data, err := ioutil.ReadFile(fmt.Sprintf("/proc/net/%s", t))
		if err != nil {
			return err
		}

		for i, l := range strings.Split(string(data), "\n") {
			if i == 0 {
				continue
			}

			f := strings.Fields(l)
			if len(f) >= 10 {
				inode, err := strconv.ParseUint(f[9], 10, 64)
				if err == nil && inode > 0 {
					srcAddr, srcPort, err := list.parse(f[1])
					if err != nil {
						continue
					}

					dstAddr, dstPort, err := list.parse(f[2])
					if err != nil {
						continue
					}

					network := DataNetwork{
						t:       t,
						srcAddr: srcAddr, srcPort: srcPort,
						dstAddr: dstAddr, dstPort: dstPort,
						isListening: dstAddr.IsUnspecified() && dstPort == 0,
					}

					list.networkByINode[inode] = network
				}
			}
		}
	}

	return nil
}

var (
	errInvalidAddress = errors.New("Invalid addresss")
)

func (list *DataNetworkList) parse(s string) (net.IP, uint16, error) {
	i := strings.IndexByte(s, ':')
	if i <= 0 {
		return nil, 0, errInvalidAddress
	}

	addr := s[:i]
	port := s[i+1:]

	l := len(addr)
	if l != 8 && l != 32 {
		return nil, 0, errInvalidAddress
	}

	valueIP, err := hex.DecodeString(addr)
	if err != nil {
		return nil, 0, err
	}

	valuePort, err := strconv.ParseUint(port, 16, 16)
	if err != nil {
		return nil, 0, err
	}

	if list.needFlip {
		list.flip(valueIP)
	}

	return valueIP, uint16(valuePort), nil
}

func (list *DataNetworkList) flip(l []uint8) {
	i := 0
	j := len(l) - 1
	for i < j {
		l[i], l[j] = l[j], l[i]
		i += 1
		j -= 1
	}
}

func getNullTerminatedString(l []byte) string {
	for i := 0; i < len(l); i += 1 {
		if l[i] == 0 {
			return string(l[0:i])
		}
	}
	return string(l)
}

func getField(s string, n int) string {
	l := strings.Fields(s)
	if len(l) > n {
		return l[n]
	}
	return ""
}

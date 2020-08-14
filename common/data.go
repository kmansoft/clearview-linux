package common

import (
	"net"
	"strings"
	"time"
)

// Network

type DataNetwork struct {
	MacAddr   string             `json:"mac_addr"`
	Combined  DataNetworkItem    `json:"combined"`
	Interface []*DataNetworkItem `json:"interface_list"`
	Ip46      DataNetworkIp46    `json:"ip4_ip6"`
}

type DataNetworkItem struct {
	Name string `json:"name"`

	RxCumulative uint64 `json:"-"`
	TxCumulative uint64 `json:"-"`

	RxInstant uint64 `json:"rx"`
	TxInstant uint64 `json:"tx"`
}

type DataNetworkIp46 struct {
	RxIP4Cumulative uint64 `json:"-"`
	TxIP4Cumulative uint64 `json:"-"`
	RxIP4Instant    uint64 `json:"ip4_rx"`
	TxIP4Instant    uint64 `json:"ip4_tx"`

	RxIP6Cumulative uint64 `json:"-"`
	TxIP6Cumulative uint64 `json:"-"`
	RxIP6Instant    uint64 `json:"ip6_rx"`
	TxIP6Instant    uint64 `json:"ip6_tx"`
}

// Memory

type DataMemory struct {
	RealUsed    uint64 `json:"real_used"`
	RealFree    uint64 `json:"real_free"`
	RealBuffers uint64 `json:"real_buffers"`
	RealCache   uint64 `json:"real_cache"`
	SwapUsed    uint64 `json:"swap_used"`
	SwapFree    uint64 `json:"swap_free"`
}

// CPU

type DataCpu struct {
	CpuCount uint64         `json:"cpu_count"`
	CpuList  []*DataCpuItem `json:"cpu_list"`
	Combined DataCpuItem    `json:"combined"`
	LoadAvg  float64        `json:"load_avg"`
}

type DataCpuItem struct {
	N uint64 `json:"n"`

	UserCumulative   uint64 `json:"-"`
	SystemCumulative uint64 `json:"-"`
	WaitCumulative   uint64 `json:"-"`

	UserInstant   uint64 `json:"user"`
	SystemInstant uint64 `json:"system"`
	WaitInstant   uint64 `json:"wait"`
}

// Disk

type DataDisk struct {
	DiskList []*DataDiskItem `json:"disk_list"`
	Combined DataDiskBasics  `json:"combined"`
}

type DataDiskBasics struct {
	ReadCountCumulative  uint64 `json:"-"`
	WriteCountCumulative uint64 `json:"-"`
	ReadCountInstant     uint64 `json:"read_count"`
	WriteCountInstant    uint64 `json:"write_count"`

	ReadBytesCumulative  uint64 `json:"-"`
	WriteBytesCumulative uint64 `json:"-"`
	ReadBytesInstant     uint64 `json:"read_bytes"`
	WriteBytesInstant    uint64 `json:"write_bytes"`
}

func (a *DataDiskBasics) Equals(b *DataDiskBasics) bool {
	return a.ReadCountInstant == b.ReadCountInstant && a.WriteCountInstant == b.WriteCountInstant &&
		a.ReadBytesInstant == b.ReadBytesInstant && a.WriteBytesInstant == b.WriteBytesInstant
}

type DataDiskItem struct {
	DataDiskBasics

	BytesFree  uint64 `json:"bytes_free"`
	BytesTotal uint64 `json:"bytes_total"`
	INodeFree  uint64 `json:"inode_free"`
	INodeTotal uint64 `json:"inode_total"`

	Name      string `json:"name"`
	Path      string `json:"path"`
	IsMounted bool   `json:"is_mounted"`
	IsSwap    bool   `json:"is_swap"`
}

func (a *DataDiskItem) Equals(b *DataDiskItem) bool {
	return a.DataDiskBasics.Equals(&b.DataDiskBasics) &&
		a.BytesFree == b.BytesFree && a.BytesTotal == b.BytesTotal &&
		a.INodeFree == b.INodeFree && a.INodeTotal == b.INodeTotal &&
		a.Name == b.Name && a.Path == b.Path &&
		a.IsMounted == b.IsMounted && a.IsSwap == b.IsSwap
}

// Processes / ports

type DataProcess struct {
	PID     uint64 `json:"-"`
	PPID    uint64 `json:"-"`
	UID     uint64 `json:"-"`
	Command string `json:"command"`
	Name    string `json:"name"`
	User    string `json:"user"`
	Count   uint64 `json:"count"`
	RSS     uint64 `json:"rss"`
	Age     uint64 `json:"age"`

	CPUCumulative uint64  `json:"-"`
	CPUInstant    uint64  `json:"cpu"`
	CPUScaled     float64 `json:"-"`

	IOReadCharsCumulative  uint64 `json:"-"`
	IOReadCharsInstant     uint64 `json:"io_read_chars"`
	IOWriteCharsCumulative uint64 `json:"-"`
	IOWriteCharsInstant    uint64 `json:"io_write_chars"`

	IOReadBytesCumulative  uint64 `json:"-"`
	IOReadBytesInstant     uint64 `json:"io_read_bytes"`
	IOWriteBytesCumulative uint64 `json:"-"`
	IOWriteBytesInstant    uint64 `json:"io_write_bytes"`
}

type DataPorts struct {
	Listen []*DataPortsListenItem `json:"listen"`
	Active []*DataPortsActiveItem `json:"active"`
}

type DataPortsListenItem struct {
	User    string `json:"user"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	SrcAddr net.IP `json:"addr"`
	SrcPort uint16 `json:"port"`
}

type DataPortsActiveItem struct {
	User  string `json:"user"`
	Name  string `json:"name"`
	Count uint64 `json:"count"`
}

type DataSystemText struct {
	CpuLabel       string `json:"cpu_label"`
	KernelLabel    string `json:"kernel_label"`
	OsDistLabel    string `json:"os_dist_label"`
	OsVersionLabel string `json:"os_version_label"`

	AppApacheVersion string `json:"app_apache_version"`
	AppNginxVersion  string `json:"app_nginx_version"`
	AppMysqlVersion  string `json:"app_mysql_version"`
	AppPgsqlVersion  string `json:"app_pgsql_version"`
}

type DataSystemMemory struct {
	RealMemorySize uint64 `json:"mem_real_size"`
	RealMemoryUsed uint64 `json:"mem_real_used"`
	SwapMemorySize uint64 `json:"mem_swap_size"`
	SwapMemoryUsed uint64 `json:"mem_swap_used"`
	DiskTotalSize  uint64 `json:"disk_total_size"`
	DiskTotalUsed  uint64 `json:"disk_total_used"`
}

type DataSystemInfo struct {
	DataSystemText
	DataSystemMemory
}

type DataAppApache struct {
	TotalAccessCountCumulative uint64 `json:"-"`
	TotalAccessCountInstant    uint64 `json:"total_access_count"`
	TotalByteCountCumulative   uint64 `json:"-"`
	TotalByteCountInstant      uint64 `json:"total_byte_count"`

	Version string `json:"version"`

	WorkersWaiting uint64 `json:"workers_waiting"`
	WorkersReading uint64 `json:"workers_reading"`
	WorkersWriting uint64 `json:"workers_writing"`
}

type DataAppNginx struct {
	TotalAccessCountCumulative uint64 `json:"-"`
	TotalAccessCountInstant    uint64 `json:"total_access_count"`

	ConnAcceptedCumulative uint64 `json:"-"`
	ConnAcceptedInstant    uint64 `json:"conn_accepted"`

	ConnHandledCumulative uint64 `json:"-"`
	ConnHandledInstant    uint64 `json:"conn_handled"`

	ConnActiveInstant uint64 `json:"conn_active"`

	Version string `json:"version"`

	WorkersWaiting uint64 `json:"workers_waiting"`
	WorkersReading uint64 `json:"workers_reading"`
	WorkersWriting uint64 `json:"workers_writing"`
}

type DataAppMysql struct {
	Version string `json:"version"`

	Cumulative map[string]uint64 `json:"-"`
	Instant    map[string]uint64 `json:"values"`
}

type DataAppPgsql struct {
	Version string `json:"version"`

	SeqScanCumulative uint64 `json:"-"`
	SeqScanInstant    uint64 `json:"seq_scan"`
	IdxScanCumulative uint64 `json:"-"`
	IdxScanInstant    uint64 `json:"idx_scan"`

	RowsSeqFetchCumulative uint64 `json:"-"`
	RowsSeqFetchInstant    uint64 `json:"rows_seq_scan_select"`
	RowsIdxFetchCumulative uint64 `json:"-"`
	RowsIdxFetchInstant    uint64 `json:"rows_idx_scan_select"`

	RowsSelectCumulative uint64 `json:"-"`
	RowsSelectInstant    uint64 `json:"rows_select"`

	RowsInsertCumulative uint64 `json:"-"`
	RowsInsertInstant    uint64 `json:"rows_insert"`

	RowsUpdateCumulative uint64 `json:"-"`
	RowsUpdateInstant    uint64 `json:"rows_update"`

	RowsDeleteCumulative uint64 `json:"-"`
	RowsDeleteInstant    uint64 `json:"rows_delete"`
}

// All together now

type Data struct {
	TicksPerSecond uint64          `json:"ticks"`
	UptimeTicks    uint64          `json:"uptime"`
	Network        *DataNetwork    `json:"network"`
	Memory         *DataMemory     `json:"memory"`
	Cpu            *DataCpu        `json:"cpu"`
	Disk           *DataDisk       `json:"disk"`
	Processes      []*DataProcess  `json:"processes"`
	Ports          *DataPorts      `json:"ports"`
	SysInfo        *DataSystemInfo `json:"sysinfo"`

	AppApache *DataAppApache `json:"app_apache,omitempty"`
	AppNginx  *DataAppNginx  `json:"app_nginx,omitempty"`
	AppMysql  *DataAppMysql  `json:"app_mysql,omitempty"`
	AppPgsql  *DataAppPgsql  `json:"app_pgsql,omitempty"`

	WhenTime       time.Time `json:"-"`
	WhenSeconds    int64     `json:"when"`
	DurationMillis uint64    `json:"duration"`
}

// Post from agent to server

type PostData struct {
	Version int64  `json:"version"`
	Node    string `json:"node"`
	Payload *Data  `json:"payload"`
}

type PostResponse struct {
	Sleep int64  `json:"sleep,omitempty"`
	Die   string `json:"die,omitempty"`
}

func (data *Data) HasProcess(pname string) bool {
	for _, p := range data.Processes {
		if pname == p.Name {
			return true
		}
	}

	return false
}

// Cumulative -> Instant values

type CumulativeToInstant struct {
	durationTime  time.Duration
	durationTicks uint64
}

func NewCumulativeToInstant(data *Data, prev *Data) CumulativeToInstant {
	durationTime := data.WhenTime.Sub(prev.WhenTime)
	durationTicks := (uint64(durationTime)*data.TicksPerSecond + uint64(time.Second/2)) / uint64(time.Second)

	// DEBUG fmt.Printf("Ticks per second is %d, duration is %s, %d ticks\n", data.TicksPerSecond, durationTime, durationTicks)

	return CumulativeToInstant{
		durationTime:  durationTime,
		durationTicks: durationTicks,
	}
}

func (ctoi *CumulativeToInstant) Calculate(curr, prev uint64) uint64 {
	if curr <= prev {
		return 0
	}

	return curr - prev
}

func (ctoi *CumulativeToInstant) CalculateDiskBasics(curr, prev *DataDiskBasics) {
	curr.ReadCountInstant =
		ctoi.Calculate(curr.ReadCountCumulative, prev.ReadCountCumulative)
	curr.WriteCountInstant =
		ctoi.Calculate(curr.WriteCountCumulative, prev.WriteCountCumulative)

	curr.ReadBytesInstant =
		ctoi.Calculate(curr.ReadBytesCumulative, prev.ReadBytesCumulative)
	curr.WriteBytesInstant =
		ctoi.Calculate(curr.WriteBytesCumulative, prev.WriteBytesCumulative)
}

func (data *Data) CalculateInstant(prev *Data) {
	ctoi := NewCumulativeToInstant(data, prev)

	// CPU

	data.Cpu.Combined.UserInstant =
		ctoi.Calculate(data.Cpu.Combined.UserCumulative, prev.Cpu.Combined.UserCumulative)
	data.Cpu.Combined.SystemInstant =
		ctoi.Calculate(data.Cpu.Combined.SystemCumulative, prev.Cpu.Combined.SystemCumulative)
	data.Cpu.Combined.WaitInstant =
		ctoi.Calculate(data.Cpu.Combined.WaitCumulative, prev.Cpu.Combined.WaitCumulative)

	if data.Cpu.CpuList != nil && len(data.Cpu.CpuList) == len(prev.Cpu.CpuList) {
		for i := range data.Cpu.CpuList {
			d := data.Cpu.CpuList[i]
			p := prev.Cpu.CpuList[i]

			d.UserInstant = ctoi.Calculate(d.UserCumulative, p.UserCumulative)
			d.SystemInstant = ctoi.Calculate(d.SystemCumulative, p.SystemCumulative)
			d.WaitInstant = ctoi.Calculate(d.WaitCumulative, p.WaitCumulative)
		}
	}

	// Network

	data.Network.Combined.TxInstant =
		ctoi.Calculate(data.Network.Combined.TxCumulative, prev.Network.Combined.TxCumulative)
	data.Network.Combined.RxInstant =
		ctoi.Calculate(data.Network.Combined.RxCumulative, prev.Network.Combined.RxCumulative)

	if data.Network.Interface != nil && len(data.Network.Interface) == len(prev.Network.Interface) {
		for i := range data.Network.Interface {
			d := data.Network.Interface[i]
			p := prev.Network.Interface[i]

			if d.Name == p.Name {
				d.TxInstant = ctoi.Calculate(d.TxCumulative, p.TxCumulative)
				d.RxInstant = ctoi.Calculate(d.RxCumulative, p.RxCumulative)
			}
		}
	}

	data.Network.Ip46.RxIP4Instant =
		ctoi.Calculate(data.Network.Ip46.RxIP4Cumulative, prev.Network.Ip46.RxIP4Cumulative)
	data.Network.Ip46.TxIP4Instant =
		ctoi.Calculate(data.Network.Ip46.TxIP4Cumulative, prev.Network.Ip46.TxIP4Cumulative)
	data.Network.Ip46.RxIP6Instant =
		ctoi.Calculate(data.Network.Ip46.RxIP6Cumulative, prev.Network.Ip46.RxIP6Cumulative)
	data.Network.Ip46.TxIP6Instant =
		ctoi.Calculate(data.Network.Ip46.TxIP6Cumulative, prev.Network.Ip46.TxIP6Cumulative)

	// Disk

	ctoi.CalculateDiskBasics(&data.Disk.Combined, &prev.Disk.Combined)
	for _, diskData := range data.Disk.DiskList {
		for _, diskPrev := range prev.Disk.DiskList {
			if diskData.Name == diskPrev.Name && diskData.IsSwap == diskPrev.IsSwap {
				ctoi.CalculateDiskBasics(&diskData.DataDiskBasics, &diskPrev.DataDiskBasics)
			}
		}
	}

	// Process

	mapProcessItem := make(map[string]*DataProcess)
	for _, p := range prev.Processes {
		key := p.Name + p.User
		mapProcessItem[key] = p
	}
	for _, p := range data.Processes {
		key := p.Name + p.User
		prev := mapProcessItem[key]
		if prev != nil {
			p.CPUInstant = ctoi.Calculate(p.CPUCumulative, prev.CPUCumulative)
			p.IOReadCharsInstant = ctoi.Calculate(p.IOReadCharsCumulative, prev.IOReadCharsCumulative)
			p.IOWriteCharsInstant = ctoi.Calculate(p.IOWriteCharsCumulative, prev.IOWriteCharsCumulative)
			p.IOReadBytesInstant = ctoi.Calculate(p.IOReadBytesCumulative, prev.IOReadBytesCumulative)
			p.IOWriteBytesInstant = ctoi.Calculate(p.IOWriteBytesCumulative, prev.IOWriteBytesCumulative)
		}
	}

	// Apache

	if data.AppApache != nil && prev.AppApache != nil {
		d := data.AppApache
		p := prev.AppApache

		d.TotalAccessCountInstant = ctoi.Calculate(d.TotalAccessCountCumulative, p.TotalAccessCountCumulative)
		d.TotalByteCountInstant =
			ctoi.Calculate(d.TotalByteCountCumulative, p.TotalByteCountCumulative)
	}

	// Nginx

	if data.AppNginx != nil && prev.AppNginx != nil {
		d := data.AppNginx
		p := prev.AppNginx

		d.TotalAccessCountInstant =
			ctoi.Calculate(d.TotalAccessCountCumulative, p.TotalAccessCountCumulative)
		d.ConnAcceptedInstant =
			ctoi.Calculate(d.ConnAcceptedCumulative, p.ConnAcceptedCumulative)
		d.ConnHandledInstant =
			ctoi.Calculate(d.ConnHandledInstant, p.ConnHandledInstant)
	}

	// MySQL

	if data.AppMysql != nil && prev.AppMysql != nil {
		d := data.AppMysql
		p := prev.AppMysql

		for k, valueCurr := range d.Cumulative {
			if valuePrev, ok := p.Cumulative[k]; ok && valueCurr >= valuePrev {
				d.Instant[k] = valueCurr - valuePrev
			} else {
				d.Instant[k] = 0
			}
		}
	}

	// PgSQL

	if data.AppPgsql != nil && prev.AppPgsql != nil {
		d := data.AppPgsql
		p := prev.AppPgsql

		d.SeqScanInstant =
			ctoi.Calculate(d.SeqScanCumulative, p.SeqScanCumulative)
		d.IdxScanInstant =
			ctoi.Calculate(d.IdxScanCumulative, p.IdxScanCumulative)

		d.RowsSeqFetchInstant =
			ctoi.Calculate(d.RowsSeqFetchCumulative, p.RowsSeqFetchCumulative)
		d.RowsIdxFetchInstant =
			ctoi.Calculate(d.RowsIdxFetchCumulative, p.RowsIdxFetchCumulative)

		d.RowsSelectInstant =
			ctoi.Calculate(d.RowsSelectCumulative, p.RowsSelectCumulative)
		d.RowsInsertInstant =
			ctoi.Calculate(d.RowsInsertCumulative, p.RowsInsertCumulative)
		d.RowsUpdateInstant =
			ctoi.Calculate(d.RowsUpdateCumulative, p.RowsUpdateCumulative)
		d.RowsDeleteInstant =
			ctoi.Calculate(d.RowsDeleteCumulative, p.RowsDeleteCumulative)
	}
}

func (data *Data) FilterProcessList() {
	list := make([]*DataProcess, 0)
	for _, item := range data.Processes {
		if needProcessItem(item) {
			list = append(list, item)
		}
	}

	// DEBUG fmt.Printf("FilterProcessList: from %d to %d\n", len(data.ProcessList), len(list))

	data.Processes = list
}

func needProcessItem(item *DataProcess) bool {
	if strings.HasPrefix(item.Name, "kworker/") ||
		strings.HasPrefix(item.Name, "cpuhp/") ||
		strings.HasPrefix(item.Name, "idle_inject/") ||
		strings.HasPrefix(item.Name, "irq/") ||
		strings.HasPrefix(item.Name, "ksoftirqd/") ||
		strings.HasPrefix(item.Name, "migration/") ||
		strings.HasPrefix(item.Name, "scsi_") {

		if item.RSS == 0 && item.CPUInstant == 0 &&
			item.IOReadCharsInstant == 0 && item.IOWriteCharsInstant == 0 &&
			item.IOReadBytesInstant == 0 && item.IOWriteBytesInstant == 0 {
			return false
		}
	}

	return true
}

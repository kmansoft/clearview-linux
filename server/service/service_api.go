package service

import (
	"clearview/common"
	"clearview/server/defs"
	"clearview/server/utils"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
)

// API - store and retrieve data

type ApiService struct {
	demoMode bool
}

const (
	CONTENT_ENCODING_GZIP = "gzip"
)

func onAgentV1Add(service *ApiService, model *ApiServerModel,
	requestData []byte, w http.ResponseWriter,
	r *http.Request, ps httprouter.Params) (int, string) {
	fmt.Printf("onAgentV1Add\n")

	// Read the request
	var rq RqApiV1Add
	err := json.Unmarshal(requestData, &rq)
	if err != nil {
		return http.StatusBadRequest, fmt.Sprintf("Error parsing data: %s", err)
	}

	payload := rq.Payload
	when := time.Unix(rq.Payload.WhenSeconds, 0)

	// Validate the node id
	count, err := model.UpdateNodeAgentAdd(rq.Node)
	if err != nil {
		return http.StatusInternalServerError, err.Error()
	} else if count < 1 {
		return http.StatusBadRequest, "invalid node id"
	}

	// Start storing data
	batch := model.NewStoreBatch(rq.Node, when)

	tickCount := float64(payload.DurationMillis*payload.TicksPerSecond) / 1000

	// Memory
	batch.NewPointUIntWithSub("mem", payload.Memory.RealFree, "real_free")
	batch.NewPointUIntWithSub("mem", payload.Memory.RealUsed, "real_used")
	batch.NewPointUIntWithSub("mem", payload.Memory.RealBuffers, "real_buffers")
	batch.NewPointUIntWithSub("mem", payload.Memory.RealCache, "real_cache")
	batch.NewPointUIntWithSub("mem", payload.Memory.SwapFree, "swap_free")
	batch.NewPointUIntWithSub("mem", payload.Memory.SwapUsed, "swap_used")

	// CPU load average
	batch.NewPointFloatWithSub("cpu", payload.Cpu.LoadAvg, "load_avg")

	batch.NewPointLatestFloat(payload.Cpu.LoadAvg, "load")
	batch.NewPointLatestUInt(payload.Cpu.CpuCount, "cpun")

	// CPU usage %%
	if tickCount > 0 {
		// Scale down by CPU count
		batch.NewPointFloatWithSub("cpu",
			float64(payload.Cpu.Combined.UserInstant)*100.0/(tickCount*float64(payload.Cpu.CpuCount)), "user")
		batch.NewPointFloatWithSub("cpu",
			float64(payload.Cpu.Combined.SystemInstant)*100.0/(tickCount*float64(payload.Cpu.CpuCount)), "system")
		batch.NewPointFloatWithSub("cpu",
			float64(payload.Cpu.Combined.WaitInstant)*100.0/(tickCount*float64(payload.Cpu.CpuCount)), "wait")

		// Scale down by CPU count
		batch.NewPointLatestFloat(float64(payload.Cpu.Combined.UserInstant+
			payload.Cpu.Combined.SystemInstant+
			payload.Cpu.Combined.WaitInstant)*100.0/(tickCount*float64(payload.Cpu.CpuCount)), "cpu")
	}

	// Network
	if payload.DurationMillis > 0 {
		batch.NewPointUIntWithSub("net",
			payload.Network.Combined.RxInstant*1000/payload.DurationMillis, "rx")
		batch.NewPointUIntWithSub("net",
			payload.Network.Combined.TxInstant*1000/payload.DurationMillis, "tx")
		batch.NewPointUIntWithSub("net",
			payload.Network.Ip46.RxIP4Instant*1000/payload.DurationMillis, "rx_4")
		batch.NewPointUIntWithSub("net",
			payload.Network.Ip46.TxIP4Instant*1000/payload.DurationMillis, "tx_4")
		batch.NewPointUIntWithSub("net",
			payload.Network.Ip46.RxIP6Instant*1000/payload.DurationMillis, "rx_6")
		batch.NewPointUIntWithSub("net",
			payload.Network.Ip46.TxIP6Instant*1000/payload.DurationMillis, "tx_6")

		batch.NewPointLatestUInt(
			(payload.Network.Combined.RxInstant+payload.Network.Combined.TxInstant)*
				1000/payload.DurationMillis, "net")
	}

	// Disk
	if payload.DurationMillis > 0 {
		batch.NewPointFloatWithSub("disk",
			float64(payload.Disk.Combined.ReadCountInstant)*1000.0/float64(payload.DurationMillis),
			"combined_read_count")
		batch.NewPointFloatWithSub("disk",
			float64(payload.Disk.Combined.WriteCountInstant)*1000.0/float64(payload.DurationMillis),
			"combined_write_count")
		batch.NewPointFloatWithSub("disk",
			float64(payload.Disk.Combined.ReadBytesInstant*1000/payload.DurationMillis),
			"combined_read_bytes")
		batch.NewPointFloatWithSub("disk",
			float64(payload.Disk.Combined.WriteBytesInstant*1000/payload.DurationMillis),
			"combined_write_bytes")

		if STORE_LISTS_IN_MONGO {
			err = model.StoreDiskListInMongo(rq.Node, payload.Disk.DiskList)
			if err != nil {
				return http.StatusInternalServerError, err.Error()
			}
		}

		for _, disk := range payload.Disk.DiskList {
			batch.NewPointFloatWithItemAndSub("disk_list",
				float64(disk.ReadCountInstant)*1000.0/float64(payload.DurationMillis),
				disk.Name, "read_count")
			batch.NewPointFloatWithItemAndSub("disk_list",
				float64(disk.WriteCountInstant)*1000.0/float64(payload.DurationMillis),
				disk.Name, "write_count")
			batch.NewPointFloatWithItemAndSub("disk_list",
				float64(disk.BytesFree),
				disk.Name, "free_bytes")
			batch.NewPointFloatWithItemAndSub("disk_list",
				float64(disk.BytesTotal),
				disk.Name, "total_bytes")
			batch.NewPointFloatWithItemAndSub("disk_list",
				float64(disk.INodeFree),
				disk.Name, "free_inodes")
			batch.NewPointFloatWithItemAndSub("disk_list",
				float64(disk.INodeTotal),
				disk.Name, "total_inodes")
		}
	}

	// Processes
	if payload.DurationMillis > 0 {
		for _, process := range payload.Processes {
			if tickCount > 0 {
				process.CPUScaled =
					float64(process.CPUInstant) * 100.0 / (tickCount * float64(payload.Cpu.CpuCount))
			} else {
				process.CPUScaled = 0.0
			}

			if payload.DurationMillis > 0 {
				process.IOReadCharsInstant = process.IOReadCharsInstant * 1000 / payload.DurationMillis
				process.IOWriteCharsInstant = process.IOWriteCharsInstant * 1000 / payload.DurationMillis
				process.IOReadBytesInstant = process.IOReadBytesInstant * 1000 / payload.DurationMillis
				process.IOWriteBytesInstant = process.IOWriteBytesInstant * 1000 / payload.DurationMillis
			} else {
				process.IOReadCharsInstant = 0
				process.IOWriteCharsInstant = 0
				process.IOReadBytesInstant = 0
				process.IOWriteBytesInstant = 0
			}
		}

		if STORE_LISTS_IN_MONGO {
			err = model.StoreProcessListInMongo(rq.Node, payload.Processes)
			if err != nil {
				return http.StatusInternalServerError, err.Error()
			}
		}

		for _, process := range payload.Processes {
			item := process.Name + PROCESS_NAME_USER_SEPARATOR + process.User

			batch.NewPointFloatWithItemAndSub("process_list",
				float64(process.Count),
				item, "instance_count")
			batch.NewPointFloatWithItemAndSub("process_list",
				float64(process.RSS),
				item, "rss")
			batch.NewPointFloatWithItemAndSub("process_list",
				float64(process.IOReadCharsInstant),
				item, "io_chars_read")
			batch.NewPointFloatWithItemAndSub("process_list",
				float64(process.IOWriteCharsInstant),
				item, "io_chars_write")
			batch.NewPointFloatWithItemAndSub("process_list",
				float64(process.IOReadBytesInstant),
				item, "io_bytes_read")
			batch.NewPointFloatWithItemAndSub("process_list",
				float64(process.IOWriteBytesInstant),
				item, "io_bytes_write")
			batch.NewPointFloatWithItemAndSub("process_list",
				process.CPUScaled,
				item, "cpu")
		}
	}

	// System info
	batch.NewPointStringWithSub("sys_text",
		payload.SysInfo.CpuLabel, "cpu")
	batch.NewPointStringWithSub("sys_text",
		payload.SysInfo.KernelLabel, "kernel")
	batch.NewPointStringWithSub("sys_text",
		payload.SysInfo.OsDistLabel, "os_name")
	batch.NewPointStringWithSub("sys_text",
		payload.SysInfo.OsVersionLabel, "os_version")

	batch.NewPointUIntWithSub("sys",
		payload.SysInfo.RealMemorySize, "realmemsize")
	batch.NewPointUIntWithSub("sys",
		payload.SysInfo.RealMemoryUsed, "realmemused")
	batch.NewPointUIntWithSub("sys",
		payload.SysInfo.SwapMemorySize, "swapsize")
	batch.NewPointUIntWithSub("sys",
		payload.SysInfo.SwapMemoryUsed, "swapused")
	batch.NewPointUIntWithSub("sys",
		payload.SysInfo.DiskTotalSize, "disksize")
	batch.NewPointUIntWithSub("sys",
		payload.SysInfo.DiskTotalUsed, "diskused")

	batch.NewPointLatestUInt(
		payload.SysInfo.RealMemorySize, "realmemsize")
	batch.NewPointLatestUInt(
		payload.SysInfo.RealMemoryUsed, "realmemused")
	batch.NewPointLatestUInt(
		payload.SysInfo.SwapMemorySize, "swapsize")
	batch.NewPointLatestUInt(
		payload.SysInfo.SwapMemoryUsed, "swapused")
	batch.NewPointLatestUInt(
		payload.SysInfo.DiskTotalSize, "disksize")
	batch.NewPointLatestUInt(
		payload.SysInfo.DiskTotalUsed, "diskused")

	if payload.SysInfo.RealMemorySize > 0 {
		batch.NewPointLatestUInt(
			payload.SysInfo.RealMemoryUsed*100/payload.SysInfo.RealMemorySize,
			"mem")
	} else {
		batch.NewPointLatestUInt(0, "mem")
	}
	if payload.SysInfo.SwapMemorySize > 0 {
		batch.NewPointLatestUInt(
			payload.SysInfo.SwapMemoryUsed*100/payload.SysInfo.SwapMemorySize,
			"swap")
	} else {
		batch.NewPointLatestUInt(0, "swap")
	}

	// App: Apache
	if payload.AppApache != nil && payload.DurationMillis > 0 {
		p := payload.AppApache

		batch.NewPointUIntWithSub("app_apache",
			p.TotalAccessCountInstant, "access")
		batch.NewPointUIntWithSub("app_apache",
			p.TotalByteCountInstant*1000/payload.DurationMillis, "bytes")

		batch.NewPointStringWithSub("app_apache_text",
			p.Version, "app_apache_version")

		batch.NewPointUIntWithSub("app_apache",
			p.WorkersWaiting, "workers_waiting")
		batch.NewPointUIntWithSub("app_apache",
			p.WorkersReading, "workers_reading")
		batch.NewPointUIntWithSub("app_apache",
			p.WorkersWriting, "workers_writing")
	} else {
		batch.NewPointStringWithSub("app_apache_text",
			"", "app_apache_version")
	}

	// App: Nginx
	if payload.AppNginx != nil && payload.DurationMillis > 0 {
		p := payload.AppNginx

		batch.NewPointUIntWithSub("app_nginx",
			p.TotalAccessCountInstant, "access")
		batch.NewPointUIntWithSub("app_nginx",
			p.ConnAcceptedInstant*1000/payload.DurationMillis, "accepted")
		batch.NewPointUIntWithSub("app_nginx",
			p.ConnHandledInstant*1000/payload.DurationMillis, "handled")

		batch.NewPointStringWithSub("app_nginx_text",
			p.Version, "app_nginx_version")

		batch.NewPointUIntWithSub("app_nginx",
			p.WorkersWaiting, "workers_waiting")
		batch.NewPointUIntWithSub("app_nginx",
			p.WorkersReading, "workers_reading")
		batch.NewPointUIntWithSub("app_nginx",
			p.WorkersWriting, "workers_writing")
	} else {
		batch.NewPointStringWithSub("app_nginx_text",
			"", "app_nginx_version")
	}

	// App: MySQL
	if payload.AppMysql != nil && payload.DurationMillis > 0 {
		p := payload.AppMysql

		for key, value := range p.Instant {
			batch.NewPointUIntWithSub("app_mysql",
				value, strings.ToLower(key))
		}

		batch.NewPointStringWithSub("app_mysql_text",
			p.Version, "app_mysql_version")
	} else {
		batch.NewPointStringWithSub("app_mysql_text",
			"", "app_mysql_version")
	}

	// App: PgSQL
	if payload.AppPgsql != nil && payload.DurationMillis > 0 {
		p := payload.AppPgsql

		batch.NewPointUIntWithSub("app_pgsql",
			p.SeqScanInstant, "seq_scan")
		batch.NewPointUIntWithSub("app_pgsql",
			p.IdxScanInstant, "idx_scan")

		batch.NewPointUIntWithSub("app_pgsql",
			p.RowsSeqFetchInstant, "rows_seq_scan_select")
		batch.NewPointUIntWithSub("app_pgsql",
			p.RowsIdxFetchInstant, "rows_idx_scan_select")

		batch.NewPointUIntWithSub("app_pgsql",
			p.RowsSelectInstant, "rows_select")
		batch.NewPointUIntWithSub("app_pgsql",
			p.RowsInsertInstant, "rows_insert")
		batch.NewPointUIntWithSub("app_pgsql",
			p.RowsUpdateInstant, "rows_update")
		batch.NewPointUIntWithSub("app_pgsql",
			p.RowsDeleteInstant, "rows_delete")

		batch.NewPointStringWithSub("app_pgsql_text",
			payload.AppPgsql.Version, "app_pgsql_version")
	} else {
		batch.NewPointStringWithSub("app_pgsql_text",
			"", "app_pgsql_version")
	}

	// Save to database
	err = batch.Save(model.influxWrite)
	if err != nil {
		fmt.Printf("Write error: %s\n", err)
	}

	// Ports - listening
	model.StorePortsListen(rq.Node, payload.Ports.Listen)

	// Ports - active
	model.StorePortsActive(rq.Node, payload.Ports.Active)

	// Create response
	rs := RsApiV1Add{Sleep: 0}
	rsData, _ := json.Marshal(&rs)

	return http.StatusOK, string(rsData)
}

func onApiV1Index(service *ApiService, model *ApiServerModel,
	requestData []byte, w http.ResponseWriter,
	r *http.Request, ps httprouter.Params) (int, string) {
	fmt.Printf("onApiV1Index\n")

	// Read the request
	var rq RqApiV1Index
	err := json.Unmarshal(requestData, &rq)
	if err != nil {
		return http.StatusBadRequest, fmt.Sprintf("Error parsing data: %s", err)
	}

	// Query nodes with "latest" data

	var rs RsApiV1Index
	rs.DemoMode = service.demoMode
	rs.NodeList = model.QueryLatestForAll()

	return utils.WriteJsonResponse(w, &rs)
}

func onApiV1GetNodeTitle(service *ApiService, model *ApiServerModel,
	requestData []byte, w http.ResponseWriter,
	r *http.Request, ps httprouter.Params) (int, string) {
	fmt.Printf("onApiV1GetNodeTitle\n")

	// Read the request
	var rq RqApiV1GetNodeTitle
	err := json.Unmarshal(requestData, &rq)
	if err != nil {
		return http.StatusBadRequest, fmt.Sprintf("Error parsing data: %s", err)
	}

	requestDataIndented, _ := json.MarshalIndent(&rq, "", "\t")
	fmt.Printf("Get data:\n%s\n", requestDataIndented)

	// Query account's nodes with "latest" data

	var rs RsApiV1GetNodeTitle
	rs.NodeId = rq.NodeId
	rs.Title, err = model.GetNodeTitle(rq.NodeId)
	if err != nil {
		return http.StatusInternalServerError, err.Error()
	}

	return utils.WriteJsonResponse(w, &rs)
}

func onApiV1SetNodeTitle(service *ApiService, model *ApiServerModel,
	requestData []byte, w http.ResponseWriter,
	r *http.Request, ps httprouter.Params) (int, string) {
	fmt.Printf("onApiV1SetNodeTitle\n")

	if service.demoMode {
		// Demo mode
		rsDemo := RsApiV1Index{
			DemoMode: true,
		}
		return utils.WriteJsonResponse(w, &rsDemo)
	}

	// Read the request
	var rq RqApiV1SetNodeTitle
	err := json.Unmarshal(requestData, &rq)
	if err != nil {
		return http.StatusBadRequest, fmt.Sprintf("Error parsing data: %s", err)
	}

	err = model.SetNodeTitle(rq.NodeId, rq.Title)
	if err != nil {
		return http.StatusInternalServerError, err.Error()
	}

	var rs RsApiV1SetNodeTitle
	rs.NodeList = model.QueryLatestForAll()

	return utils.WriteJsonResponse(w, &rs)
}

func onApiV1CreateNode(service *ApiService, model *ApiServerModel,
	requestData []byte, w http.ResponseWriter,
	r *http.Request, ps httprouter.Params) (int, string) {
	fmt.Printf("onApiV1CreateNode\n")

	if service.demoMode {
		// Demo mode
		count, err := model.GetNodeCount()
		if err != nil {
			return http.StatusInternalServerError, err.Error()
		} else if count >= 1 {
			rsDemo := RsApiV1Index{
				DemoMode: true,
			}
			return utils.WriteJsonResponse(w, &rsDemo)
		}
	}

	// Read the request
	var rq RqApiV1CreateNode
	err := json.Unmarshal(requestData, &rq)
	if err != nil {
		return http.StatusBadRequest, fmt.Sprintf("Error parsing data: %s", err)
	}

	nodeId, err := model.CreateNode()
	if err != nil {
		return http.StatusInternalServerError, err.Error()
	}

	var rs RsApiV1CreateNode
	rs.NewNodeId = nodeId
	rs.NodeList = model.QueryLatestForAll()

	return utils.WriteJsonResponse(w, &rs)
}

func onApiV1DeleteNode(service *ApiService, model *ApiServerModel,
	requestData []byte, w http.ResponseWriter,
	r *http.Request, ps httprouter.Params) (int, string) {
	fmt.Printf("onApiV1DeleteNode\n")

	if service.demoMode {
		// Demo mode
		rsDemo := RsApiV1Index{
			DemoMode: true,
		}
		return utils.WriteJsonResponse(w, &rsDemo)
	}

	// Read the request
	var rq RqApiV1DeleteNode
	err := json.Unmarshal(requestData, &rq)
	if err != nil {
		return http.StatusBadRequest, fmt.Sprintf("Error parsing data: %s", err)
	}

	err = model.DeleteNode(rq.NodeId)
	if err != nil {
		return http.StatusInternalServerError, err.Error()
	}

	var rs RsApiV1DeleteNode
	rs.NodeList = model.QueryLatestForAll()

	return utils.WriteJsonResponse(w, &rs)
}

func onApiV1Get(service *ApiService, model *ApiServerModel,
	requestData []byte, w http.ResponseWriter,
	r *http.Request, ps httprouter.Params) (int, string) {
	fmt.Printf("onApiV1Get\n")

	// Read the request
	var rq RqApiV1Get
	err := json.Unmarshal(requestData, &rq)
	if err != nil {
		return http.StatusBadRequest, fmt.Sprintf("Error parsing data: %s", err)
	}

	rqCommon := rq.Normalize()

	list := model.QueryDataGet(defs.TECH_INFLUX, rqCommon, rq.Series, rq.Item)

	rs := RsApiV1Get{
		Rq:     rq,
		Series: list,
	}
	rsBytes, err := json.Marshal(&rs)
	if err != nil {
		return http.StatusInternalServerError, err.Error()
	}

	if utils.WriteResponseWithCompression(w, r, rsBytes) {
		return 0, ""
	}

	return http.StatusOK, string(rsBytes)
}

func onApiV1DiskOverview(service *ApiService, model *ApiServerModel,
	requestData []byte, w http.ResponseWriter,
	r *http.Request, ps httprouter.Params) (int, string) {
	fmt.Printf("onApiV1DiskOverview\n")

	// Read the request
	var rq RqApiV1DiskOverview
	err := json.Unmarshal(requestData, &rq)
	if err != nil {
		return http.StatusBadRequest, fmt.Sprintf("Error parsing data: %s", err)
	}

	requestDataIndented, _ := json.MarshalIndent(&rq, "", "\t")
	fmt.Printf("Get data:\n%s\n", requestDataIndented)

	rqCommon := rq.Normalize()

	list := model.QueryDataDiskOverview(defs.TECH_INFLUX, rqCommon)

	rs := RsApiV1DiskOverview{
		Rq:    rq,
		Disks: list,
	}
	rsBytes, err := json.Marshal(&rs)
	if err != nil {
		return http.StatusInternalServerError, err.Error()
	}

	if utils.WriteResponseWithCompression(w, r, rsBytes) {
		return 0, ""
	}

	return http.StatusOK, string(rsBytes)
}

func onApiV1ProcessOverview(service *ApiService, model *ApiServerModel,
	requestData []byte, w http.ResponseWriter,
	r *http.Request, ps httprouter.Params) (int, string) {
	fmt.Printf("onApiV1ProcessOverview\n")

	// Read the request
	var rq RqApiV1ProcessOverview
	err := json.Unmarshal(requestData, &rq)
	if err != nil {
		return http.StatusBadRequest, fmt.Sprintf("Error parsing data: %s", err)
	}

	requestDataIndented, _ := json.MarshalIndent(&rq, "", "\t")
	fmt.Printf("Get data:\n%s\n", requestDataIndented)

	rqCommon := rq.Normalize()

	list := model.QueryDataProcessOverview(defs.TECH_INFLUX, rqCommon)

	rs := RsApiV1ProcessOverview{
		Rq:        rq,
		Processes: list,
	}
	rsBytes, err := json.Marshal(&rs)
	if err != nil {
		return http.StatusInternalServerError, err.Error()
	}

	if utils.WriteResponseWithCompression(w, r, rsBytes) {
		return 0, ""
	}

	return http.StatusOK, string(rsBytes)
}

func onApiV1SystemOverview(service *ApiService, model *ApiServerModel,
	requestData []byte, w http.ResponseWriter,
	r *http.Request, ps httprouter.Params) (int, string) {
	fmt.Printf("onApiV1SystemOverview\n")

	// Read the request
	var rq RqApiV1SystemOverview
	err := json.Unmarshal(requestData, &rq)
	if err != nil {
		return http.StatusBadRequest, fmt.Sprintf("Error parsing data: %s", err)
	}

	requestDataIndented, _ := json.MarshalIndent(&rq, "", "\t")
	fmt.Printf("Get data:\n%s\n", requestDataIndented)

	rqCommon := rq.Normalize()

	rs := RsApiV1SystemOverview{
		Rq:            rq,
		SystemText:    model.QuerySystemText(rqCommon.NodeId),
		SystemNumeric: &RsApiV1NodeNumeric{},
		Memory:        &common.DataSystemMemory{},
	}
	model.QuerySystemNumericAndMemory(rqCommon.NodeId, rs.SystemNumeric, rs.Memory)

	listen, active := model.QueryPorts(rq.NodeId)
	if len(listen) > 0 || len(active) > 0 {
		rs.Ports = &RsApiV1SystemOverviewPorts{
			Listen: listen, Active: active,
		}
	}

	rsBytes, err := json.Marshal(&rs)
	if err != nil {
		return http.StatusInternalServerError, err.Error()
	}

	if utils.WriteResponseWithCompression(w, r, rsBytes) {
		return 0, ""
	}

	return http.StatusOK, string(rsBytes)
}

func onApiV1AppApache(service *ApiService, model *ApiServerModel,
	requestData []byte, w http.ResponseWriter,
	r *http.Request, ps httprouter.Params) (int, string) {
	fmt.Printf("onApiV1AppApache\n")

	// Read the request
	var rq RqApiV1AppApache
	err := json.Unmarshal(requestData, &rq)
	if err != nil {
		return http.StatusBadRequest, fmt.Sprintf("Error parsing data: %s", err)
	}

	requestDataIndented, _ := json.MarshalIndent(&rq, "", "\t")
	fmt.Printf("Get data:\n%s\n", requestDataIndented)

	rqCommon := rq.Normalize()

	rs := RsApiV1AppApache{
		Rq:      rq,
		Series:  model.QueryDataGet(defs.TECH_INFLUX, rqCommon, []string{"app_apache"}, ""),
		Version: model.QueryApacheVersion(rq.NodeId),
	}

	rsBytes, err := json.Marshal(&rs)
	if err != nil {
		return http.StatusInternalServerError, err.Error()
	}

	if utils.WriteResponseWithCompression(w, r, rsBytes) {
		return 0, ""
	}

	return http.StatusOK, string(rsBytes)
}

func onApiV1AppNginx(service *ApiService, model *ApiServerModel,
	requestData []byte, w http.ResponseWriter,
	r *http.Request, ps httprouter.Params) (int, string) {
	fmt.Printf("onApiV1AppNginx\n")

	// Read the request
	var rq RqApiV1AppNginx
	err := json.Unmarshal(requestData, &rq)
	if err != nil {
		return http.StatusBadRequest, fmt.Sprintf("Error parsing data: %s", err)
	}

	requestDataIndented, _ := json.MarshalIndent(&rq, "", "\t")
	fmt.Printf("Get data:\n%s\n", requestDataIndented)

	rqCommon := rq.Normalize()

	rs := RsApiV1AppNginx{
		Rq:      rq,
		Series:  model.QueryDataGet(defs.TECH_INFLUX, rqCommon, []string{"app_nginx"}, ""),
		Version: model.QueryNginxVersion(rq.NodeId),
	}

	rsBytes, err := json.Marshal(&rs)
	if err != nil {
		return http.StatusInternalServerError, err.Error()
	}

	if utils.WriteResponseWithCompression(w, r, rsBytes) {
		return 0, ""
	}

	return http.StatusOK, string(rsBytes)
}

func onApiV1AppMysql(service *ApiService, model *ApiServerModel,
	requestData []byte, w http.ResponseWriter,
	r *http.Request, ps httprouter.Params) (int, string) {
	fmt.Printf("onApiV1AppMysql\n")

	// Read the request
	var rq RqApiV1AppMysql
	err := json.Unmarshal(requestData, &rq)
	if err != nil {
		return http.StatusBadRequest, fmt.Sprintf("Error parsing data: %s", err)
	}

	requestDataIndented, _ := json.MarshalIndent(&rq, "", "\t")
	fmt.Printf("Get data:\n%s\n", requestDataIndented)

	rqCommon := rq.Normalize()

	rs := RsApiV1AppMysql{
		Rq:      rq,
		Series:  model.QueryDataGet(defs.TECH_INFLUX, rqCommon, []string{"app_mysql"}, ""),
		Version: model.QueryMysqlVersion(rq.NodeId),
	}

	rsBytes, err := json.Marshal(&rs)
	if err != nil {
		return http.StatusInternalServerError, err.Error()
	}

	if utils.WriteResponseWithCompression(w, r, rsBytes) {
		return 0, ""
	}

	return http.StatusOK, string(rsBytes)
}

func onApiV1AppPgsql(service *ApiService, model *ApiServerModel,
	requestData []byte, w http.ResponseWriter,
	r *http.Request, ps httprouter.Params) (int, string) {
	fmt.Printf("onApiV1AppPgsql\n")

	// Read the request
	var rq RqApiV1AppPgsql
	err := json.Unmarshal(requestData, &rq)
	if err != nil {
		return http.StatusBadRequest, fmt.Sprintf("Error parsing data: %s", err)
	}

	requestDataIndented, _ := json.MarshalIndent(&rq, "", "\t")
	fmt.Printf("Get data:\n%s\n", requestDataIndented)

	rqCommon := rq.Normalize()

	rs := RsApiV1AppPgsql{
		Rq:      rq,
		Series:  model.QueryDataGet(defs.TECH_INFLUX, rqCommon, []string{"app_pgsql"}, ""),
		Version: model.QueryPgsqlVersion(rq.NodeId),
	}

	rsBytes, err := json.Marshal(&rs)
	if err != nil {
		return http.StatusInternalServerError, err.Error()
	}

	if utils.WriteResponseWithCompression(w, r, rsBytes) {
		return 0, ""
	}

	return http.StatusOK, string(rsBytes)
}

type ApiModelFuncPost func(*ApiService, *ApiServerModel,
	[]byte, http.ResponseWriter, *http.Request, httprouter.Params) (int, string)

func makeApiModelFunc(service *ApiService, model *ApiServerModel,
	handle ApiModelFuncPost) httprouter.Handle {

	return func(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
		now0 := utils.StartHandlingHttpRequest(w, req)

		// Check content type
		if req.Body == nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		ctype := req.Header.Get(utils.CONTENT_TYPE)
		sep := strings.IndexByte(ctype, ';')
		if sep != -1 {
			ctype = ctype[:sep]
		}

		if ctype != utils.REQ_MIME_APPLICATION_JSON {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Read the request
		requestData, err := ioutil.ReadAll(io.LimitReader(req.Body, utils.MAX_HTTP_REQUEST_SIZE))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Decompress if compressed
		if req.Header.Get(utils.CONTENT_ENCODING) == CONTENT_ENCODING_GZIP {
			decompressedData, err := utils.ReadRequestWithDecompression(requestData)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			requestData = decompressedData
		}

		// Call the handler
		status, responseData := handle(service, model, requestData, w, req, params)

		// Send the response
		utils.FinishHandlingHttpRequest(w, req, status, responseData, now0, utils.RESP_MIME_APPLICATION_JSON_UTF_8)
	}
}

func NewApiService(demoMode bool) *ApiService {
	return &ApiService{
		demoMode: demoMode,
	}
}

func (service *ApiService) Setup(router *httprouter.Router, model *ApiServerModel) {

	// The agent

	router.POST("/cv/agent/v1/add",
		makeApiModelFunc(service, model, onAgentV1Add))

	// The client API

	router.POST("/cv/api/v1/index",
		makeApiModelFunc(service, model, onApiV1Index))

	router.POST("/cv/api/v1/getnodetitle",
		makeApiModelFunc(service, model, onApiV1GetNodeTitle))

	router.POST("/cv/api/v1/setnodetitle",
		makeApiModelFunc(service, model, onApiV1SetNodeTitle))

	router.POST("/cv/api/v1/createnode",
		makeApiModelFunc(service, model, onApiV1CreateNode))

	router.POST("/cv/api/v1/deletenode",
		makeApiModelFunc(service, model, onApiV1DeleteNode))

	router.POST("/cv/api/v1/get",
		makeApiModelFunc(service, model, onApiV1Get))

	router.POST("/cv/api/v1/disk_overview",
		makeApiModelFunc(service, model, onApiV1DiskOverview))

	router.POST("/cv/api/v1/process_overview",
		makeApiModelFunc(service, model, onApiV1ProcessOverview))

	router.POST("/cv/api/v1/system_overview",
		makeApiModelFunc(service, model, onApiV1SystemOverview))

	router.POST("/cv/api/v1/app_apache",
		makeApiModelFunc(service, model, onApiV1AppApache))

	router.POST("/cv/api/v1/app_nginx",
		makeApiModelFunc(service, model, onApiV1AppNginx))

	router.POST("/cv/api/v1/app_mysql",
		makeApiModelFunc(service, model, onApiV1AppMysql))

	router.POST("/cv/api/v1/app_pgsql",
		makeApiModelFunc(service, model, onApiV1AppPgsql))
}

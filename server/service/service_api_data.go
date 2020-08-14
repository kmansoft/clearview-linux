package service

import (
	"clearview/common"
	"fmt"
	"time"
)

type RqApiV1Add = common.PostData
type RsApiV1Add = common.PostResponse

type RqCommon struct {
	NodeId        string `json:"node_id"`
	EndTime       int64  `json:"end_time"`
	PointCount    uint32 `json:"point_count"`
	PointDuration uint32 `json:"point_duration"` // In seconds
}

type RqCommonModel struct {
	NodeId        string
	StartTime     time.Time
	EndTime       time.Time
	PointCount    uint32
	PointDuration uint32
}

func (rq *RqCommon) Normalize() RqCommonModel {
	if rq.EndTime == 0 {
		rq.EndTime = time.Now().Unix()
	}
	if rq.PointCount <= 0 || rq.PointCount >= 90 {
		rq.PointCount = 60
	}
	if rq.PointDuration <= 0 {
		rq.PointDuration = 60
	}

	end := rq.EndTime
	start := end - (int64(rq.PointCount) * int64(rq.PointDuration))

	startTime := time.Unix(start, 0)
	endTime := time.Unix(end, 0)

	fmt.Printf("From %s to %s\n", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))

	return RqCommonModel{NodeId: rq.NodeId,
		StartTime: startTime, EndTime: endTime,
		PointCount: rq.PointCount, PointDuration: rq.PointDuration}
}

type RqApiV1GetNodeTitle struct {
	NodeId string `json:"node_id"`
}

type RsApiV1GetNodeTitle struct {
	NodeId string `json:"node_id"`
	Title  string `json:"node_title"`
}

type RqApiV1Index struct {
}

type RsApiV1Index struct {
	DemoMode bool                  `json:"demo_mode"`
	NodeList []*RsApiV1NodeNumeric `json:"node_list"`
}

type RsApiV1NodeNumeric struct {
	Account      string  `json:"account_id"`
	Node         string  `json:"node_id"`
	Title        string  `json:"node_title"`
	ValueCpu     float64 `json:"value_cpu"`
	ValueMemory  uint64  `json:"value_memory"`
	ValueSwap    uint64  `json:"value_swap"`
	ValueLoad    float64 `json:"value_load"`
	ValueNetwork uint64  `json:"value_network"`
	ValueCpuN    uint64  `json:"value_cpun"`
	When         uint64  `json:"when"`
}

type RqApiV1SetNodeTitle struct {
	NodeId string `json:"node_id"`
	Title  string `json:"node_title"`
}

type RsApiV1SetNodeTitle struct {
	RsApiV1Index
}

type RqApiV1CreateNode struct {
}

type RsApiV1CreateNode struct {
	RsApiV1Index
	NewNodeId string `json:"new_node_id"`
}

type RqApiV1DeleteNode struct {
	NodeId string `json:"node_id"`
}

type RsApiV1DeleteNode struct {
	RsApiV1Index
}

type RqApiV1Get struct {
	Series []string `json:"series"`
	Item   string   `json:"item"`
	RqCommon
}

type RsApiV1Get struct {
	Rq     RqApiV1Get         `json:"request"`
	Series []RsApiV1GetSeries `json:"series"`
}

type RsApiV1GetSeries struct {
	SubValue string            `json:"sub"`
	Points   []RsApiV1GetPoint `json:"points"`
}

type RsApiV1GetPoint struct {
	PointTime  int64   `json:"t"`
	PointValue float64 `json:"v"`
	IsValueNil bool    `json:"n,omitempty"`
}

type RqApiV1DiskOverview struct {
	RqCommon
}

type RsApiV1DiskOverview struct {
	Rq    RqApiV1DiskOverview        `json:"request"`
	Disks []*RsApiV1DiskOverviewItem `json:"disks"`
}

type RqApiV1ProcessOverview struct {
	RqCommon
}

type RsApiV1DiskOverviewItem struct {
	Name       string `json:"name"`
	ReadOps    uint64 `json:"read_ops"`
	WriteOps   uint64 `json:"write_ops"`
	SpaceFree  uint64 `json:"space_free"`
	SpaceTotal uint64 `json:"space_total"`
	INodeFree  uint64 `json:"inode_free"`
	INodeTotal uint64 `json:"inode_total"`
}

type RsApiV1ProcessOverview struct {
	Rq        RqApiV1ProcessOverview        `json:"request"`
	Processes []*RsApiV1ProcessOverviewItem `json:"processes"`
}

type RsApiV1ProcessOverviewItem struct {
	Name         string  `json:"name"`
	User         string  `json:"user"`
	Count        uint64  `json:"count"`
	Memory       uint64  `json:"memory"`
	IOCharsRead  uint64  `json:"-"`
	IOCharsWrite uint64  `json:"-"`
	IOBytesRead  uint64  `json:"-"`
	IOBytesWrite uint64  `json:"-"`
	IOTotal      uint64  `json:"io_total"`
	CPU          float64 `json:"cpu"`
}

type RqApiV1SystemOverview struct {
	RqCommon
}

type RsApiV1SystemOverview struct {
	Rq            RqApiV1SystemOverview       `json:"request"`
	SystemText    *common.DataSystemText      `json:"system_text"`
	SystemNumeric *RsApiV1NodeNumeric         `json:"system_numeric"`
	Memory        *common.DataSystemMemory    `json:"memory"`
	Ports         *RsApiV1SystemOverviewPorts `json:"ports"`
}

type RsApiV1SystemOverviewPorts struct {
	Listen []*SyncPortListen `json:"listen"`
	Active []*SyncPortActive `json:"active"`
}

type RqApiV1AppApache struct {
	RqCommon
}

type RsApiV1AppApache struct {
	Rq      RqApiV1AppApache   `json:"request"`
	Series  []RsApiV1GetSeries `json:"series"`
	Version string             `json:"version"`
}

type RqApiV1AppNginx struct {
	RqCommon
}

type RsApiV1AppNginx struct {
	Rq      RqApiV1AppNginx    `json:"request"`
	Series  []RsApiV1GetSeries `json:"series"`
	Version string             `json:"version"`
}

type RqApiV1AppMysql struct {
	RqCommon
}

type RsApiV1AppMysql struct {
	Rq      RqApiV1AppMysql    `json:"request"`
	Series  []RsApiV1GetSeries `json:"series"`
	Version string             `json:"version"`
}

type RqApiV1AppPgsql struct {
	RqCommon
}

type RsApiV1AppPgsql struct {
	Rq      RqApiV1AppPgsql    `json:"request"`
	Series  []RsApiV1GetSeries `json:"series"`
	Version string             `json:"version"`
}

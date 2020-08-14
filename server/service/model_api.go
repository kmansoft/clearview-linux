package service

import (
	"clearview/common"
	"clearview/server/defs"
	"clearview/server/tsdb"
	"clearview/server/utils"
	"context"
	"encoding/json"
	"fmt"
	"github.com/pborman/uuid"
	"math"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	influx "github.com/orourkedd/influxdb1-client"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	DATABASE_URI_INFLUX   = "http://localhost:8086"
	DATABASE_SERVER_MONGO = "localhost"
	DATABASE_NAME_INFLUX  = "clearview"

	PROCESS_NAME_USER_SEPARATOR = "|"

	DATABASE_NAME_MONGO = "clearview"

	STORE_LISTS_IN_MONGO   = true
	MONGO_LIST_TIME_UPDATE = 10 * 60
	MONGO_LIST_TIME_EXPIRE = 60 * 60
)

var (
	RANDOM_EPITETS = []string{"brave", "cowardly", "tiny", "careless",
		"sleepy", "peculiar", "wise", "lively", "vivid", "boring"}
	RANDOM_ANIMALS = []string{"frog", "spider", "pigeon", "parrot",
		"turkey", "goldfish", "shark", "shrimp", "dragonfly", "ladybug",
		"beaver", "camel", "dolphin", "mouse", "crocodile", "turtle"}
)

type ApiServerModel struct {
	dbNameInflux, dbNameMongo string

	influxWrite tsdb.DbProtocolWrite
	influxRead  *influx.Client

	monConn             *mongo.Client
	monDb               *mongo.Database
	monCollNode         *mongo.Collection
	monCollValueText    *mongo.Collection
	monCollValueNumeric *mongo.Collection
	monCollPortListen   *mongo.Collection
	monCollPortActive   *mongo.Collection
	monCollDisk         *mongo.Collection
	monCollProcess      *mongo.Collection
}

func NewApiServerModel(flags defs.Flags) (*ApiServerModel, error) {
	// InfluxDB - write and read
	influxUrl, err := url.Parse(flags.InfluxDbUri)
	if err != nil {
		fmt.Printf("cannot parse uri for InfluxDB: %s\n", err)
		return nil, err
	}

	influxWriteClient, err := tsdb.NewDbProtocolWriteInflux(*influxUrl, flags.InfluxDbName)
	if err != nil {
		fmt.Printf("cannot connect to InfluxDB: %s\n", err)
		return nil, err
	}

	influxReadClient, _ := influx.NewClient(
		influx.Config{
			URL: *influxUrl,
		})

	fmt.Printf("API model - connected to InfluxDB\n")

	// Mongo
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mongoConn, err := mongo.Connect(ctx, &options.ClientOptions{
		Hosts: []string{flags.MongoServerName}})

	if err != nil {
		fmt.Printf("Cannot connect to MongoDB: %s\n", err)
		return nil, err
	}

	fmt.Printf("API model - connected to MongoDB\n")

	mongoDatabase := mongoConn.Database(flags.MongoDbName)

	mongoCollNode := mongoDatabase.Collection("node")

	// Node collection
	indexListNode, err := utils.NewIndexList(mongoCollNode)
	if err != nil {
		return nil, err
	}

	err = indexListNode.EnsureSimpleIndex("node_id", true)
	if err != nil {
		return nil, err
	}

	// Value_text and value_numeric collections
	mongoCollValueText := mongoDatabase.Collection("value_text")
	mongoCollValueNumeric := mongoDatabase.Collection("value_numeric")
	for _, coll := range []*mongo.Collection{
		mongoCollValueText, mongoCollValueNumeric,
	} {
		indexList, err := utils.NewIndexList(coll)
		if err != nil {
			return nil, err
		}

		err = indexList.EnsureSimpleIndex("node_id", false)
		if err != nil {
			return nil, err
		}
	}

	// Other value collections
	mongoCollPortListen := mongoDatabase.Collection("port_listen")
	mongoCollPortActive := mongoDatabase.Collection("port_active")
	mongoCollDisk := mongoDatabase.Collection("disk_list")
	mongoCollProcess := mongoDatabase.Collection("process_list")
	for _, coll := range []*mongo.Collection{
		mongoCollPortListen, mongoCollPortActive, mongoCollDisk, mongoCollProcess,
	} {
		indexList, err := utils.NewIndexList(coll)
		if err != nil {
			return nil, err
		}

		err = indexList.EnsureSimpleIndex("node_id", false)
		if err != nil {
			return nil, err
		}
	}

	return &ApiServerModel{
		dbNameInflux: flags.InfluxDbName,
		dbNameMongo:  flags.MongoDbName,

		influxWrite: influxWriteClient,
		influxRead:  influxReadClient,

		monConn:             mongoConn,
		monDb:               mongoDatabase,
		monCollNode:         mongoCollNode,
		monCollValueText:    mongoCollValueText,
		monCollValueNumeric: mongoCollValueNumeric,
		monCollPortListen:   mongoCollPortListen,
		monCollPortActive:   mongoCollPortActive,
		monCollDisk:         mongoCollDisk,
		monCollProcess:      mongoCollProcess,
	}, nil
}

func (model *ApiServerModel) GetNodeCount() (int64, error) {
	count, err := model.monCollNode.CountDocuments(context.Background(), bson.M{})
	return count, err
}

func (model *ApiServerModel) GetNodeTitle(nodeId string) (string, error) {
	filter := bson.M{"node_id": nodeId}

	res := model.monCollNode.FindOne(context.Background(), filter, &options.FindOneOptions{})
	if res.Err() != nil {
		return "", res.Err()
	}

	var node LoadNode
	err := res.Decode(&node)
	if err != nil {
		return "", err
	}

	return node.Title, nil
}

func (model *ApiServerModel) SetNodeTitle(nodeId string, title string) error {
	filter := bson.M{"node_id": nodeId}
	update := bson.M{
		"$set": bson.M{
			"node_title": title,
		},
	}

	_, err := model.monCollNode.UpdateOne(context.Background(), filter, update, &options.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (model *ApiServerModel) CreateNode() (string, error) {
	nodeId := uuid.New()
	title := utils.RandomChoice(RANDOM_EPITETS) + " " + utils.RandomChoice(RANDOM_ANIMALS)

	values := bson.M{
		"node_id":    nodeId,
		"node_title": title,
	}

	_, err := model.monCollNode.InsertOne(context.Background(), values, &options.InsertOneOptions{})
	if err != nil {
		return "", fmt.Errorf("cannot insert node into mongo: %v", err)
	}

	return nodeId, nil
}

func (model *ApiServerModel) DeleteNode(nodeId string) error {
	filter := bson.M{
		"node_id": nodeId,
	}

	_, err := model.monCollNode.DeleteOne(context.Background(), filter, &options.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("cannot delete node from mongo: %v", err)
	}

	return nil
}

type StoreBatch struct {
	model         *ApiServerModel
	account, node string
	tags          map[string]string
	when          time.Time
	points        []influx.Point
	texts         []StoreText
	latestUInt    []StoreLatestUInt
	latestFloat   []StoreLatestFloat
}

type StoreText struct {
	Item  string
	Sub   string
	Value string
}

type LoadText struct {
	Item  string `bson:"item"`
	Sub   string `bson:"sub"`
	Value string `bson:"value"`
}

type StoreLatestUInt struct {
	Value uint64
	Sub   string
}

type StoreLatestFloat struct {
	Value float64
	Sub   string
}

type LoadNode struct {
	Id    primitive.ObjectID `bson:"_id"`
	Node  string             `bson:"node_id" json:"node_id"`
	Title string             `bson:"node_title" json:"node_title"`
}

type LoadLatestNumericItem struct {
	Node       string  `bson:"node_id"`
	Sub        string  `bson:"sub"`
	When       uint64  `bson:"when"`
	ValueUInt  uint64  `bson:"value_uint"`
	ValueFloat float64 `bson:"value_float"`
}

type LoadDisk struct {
	Id   primitive.ObjectID `bson:"_id"`
	Name string             `bson:"name"`
	When int64              `bson:"when"`

	ReadCount   uint64 `bson:"read_count"`
	WriteCount  uint64 `bson:"write_count"`
	FreeBytes   uint64 `bson:"free_bytes"`
	TotalBytes  uint64 `bson:"total_bytes"`
	FreeINodes  uint64 `bson:"free_inodes"`
	TotalINodes uint64 `bson:"total_inodes"`
}

func (a *LoadDisk) Equals(b *common.DataDiskItem) bool {
	return a.Name == b.Name &&
		a.ReadCount == b.ReadCountInstant && a.WriteCount == b.WriteCountInstant &&
		a.FreeBytes == b.BytesFree && a.TotalBytes == b.BytesTotal &&
		a.FreeINodes == b.INodeFree && a.TotalINodes == b.INodeTotal
}

type LoadProcess struct {
	Id   primitive.ObjectID `bson:"_id"`
	Name string             `bson:"name"`
	When int64              `bson:"when"`

	InstanceCount uint64  `bson:"instance_count"`
	Rss           uint64  `bson:"rss"`
	Cpu           float64 `bson:"cpu"`
	CharsRead     uint64  `bson:"chars_read"`
	CharsWrite    uint64  `bson:"chars_write"`
	BytesRead     uint64  `bson:"bytes_read"`
	BytesWrite    uint64  `bson:"bytes_write"`
}

func (a *LoadProcess) Equals(b *common.DataProcess, nameAndUser string) bool {
	return a.Name == nameAndUser &&
		a.InstanceCount == b.Count && a.Rss == b.RSS && a.Cpu == b.CPUScaled &&
		a.CharsRead == b.IOReadCharsInstant && a.CharsWrite == b.IOWriteCharsInstant &&
		a.BytesRead == b.IOReadBytesInstant && a.BytesWrite == b.IOWriteBytesInstant
}

func (model *ApiServerModel) UpdateNodeAgentAdd(nodeId string) (int64, error) {
	filter := bson.M{"node_id": nodeId}
	update := bson.M{
		"$inc": bson.M{
			"agent_add_count": 1,
		},
	}

	res, err := model.monCollNode.UpdateOne(context.Background(), filter, update, &options.UpdateOptions{})
	if err != nil {
		return 0, err
	}
	return res.ModifiedCount, nil
}

func (model *ApiServerModel) NewStoreBatch(node string, when time.Time) StoreBatch {
	return StoreBatch{
		model: model,
		node:  node,
		tags: map[string]string{
			"node_id": node,
		},
		when:   when,
		points: make([]influx.Point, 0),
	}
}

func (batch *StoreBatch) NewPointUInt(name string, value uint64) {
	batch.points = append(batch.points, influx.Point{
		Measurement: name,
		Tags:        batch.tags,
		Fields: map[string]interface{}{
			"value": int64(value),
		},
		// Time: time.Now(),
		// Time:      batch.when,
		Precision: "s",
	})
}

func (batch *StoreBatch) NewPointFloat(name string, value float64) {
	rounded := batch.roundValue(value)

	batch.points = append(batch.points, influx.Point{
		Measurement: name,
		Tags:        batch.tags,
		Fields: map[string]interface{}{
			"value": rounded,
		},
		// Time: time.Now(),
		// Time:      batch.when.Round(time.Second),
		Precision: "s",
	})
}

func (batch *StoreBatch) NewPointUIntWithSub(name string, value uint64, sub string) {
	tags := map[string]string{
		"node_id": batch.node,
		"sub":     sub,
	}

	batch.points = append(batch.points, influx.Point{
		Measurement: name,
		Tags:        tags,
		Fields: map[string]interface{}{
			"value": int64(value),
		},
		// Time: time.Now(),
		// Time:      batch.when,
		Precision: "s",
	})
}

func (batch *StoreBatch) NewPointFloatWithSub(name string, value float64, sub string) {
	tags := map[string]string{
		"node_id": batch.node,
		"sub":     sub,
	}

	rounded := batch.roundValue(value)

	batch.points = append(batch.points, influx.Point{
		Measurement: name,
		Tags:        tags,
		Fields: map[string]interface{}{
			"value": rounded,
		},
		// Time: time.Now(),
		// Time:      batch.when.Round(time.Second),
		Precision: "s",
	})
}

func (batch *StoreBatch) NewPointStringWithSub(name string, value string, sub string) {
	batch.texts = append(batch.texts, StoreText{
		Item:  "",
		Sub:   sub,
		Value: value,
	})
}

func (batch *StoreBatch) NewPointUIntWithItemAndSub(name string, value uint64, item string, sub string) {
	tags := map[string]string{
		"node_id": batch.node,
		"item":    item,
		"sub":     sub,
	}

	batch.points = append(batch.points, influx.Point{
		Measurement: name,
		Tags:        tags,
		Fields: map[string]interface{}{
			"value": int64(value),
		},
		// Time: time.Now(),
		// Time:      batch.when,
		Precision: "s",
	})
}

func (batch *StoreBatch) NewPointFloatWithItemAndSub(name string, value float64, item string, sub string) {
	tags := map[string]string{
		"node_id": batch.node,
		"item":    item,
		"sub":     sub,
	}

	rounded := batch.roundValue(value)

	batch.points = append(batch.points, influx.Point{
		Measurement: name,
		Tags:        tags,
		Fields: map[string]interface{}{
			"value": rounded,
		},
		// Time: time.Now(),
		// Time:      batch.when.Round(time.Second),
		Precision: "s",
	})
}

func (batch *StoreBatch) NewPointStringWithItemAndSub(name string, value string, item string, sub string) {
	batch.texts = append(batch.texts, StoreText{
		Item:  item,
		Sub:   sub,
		Value: value,
	})
}

func (batch *StoreBatch) NewPointLatestUInt(value uint64, sub string) {
	batch.latestUInt = append(batch.latestUInt, StoreLatestUInt{
		Value: value,
		Sub:   sub,
	})
}

func (batch *StoreBatch) NewPointLatestFloat(value float64, sub string) {
	batch.latestFloat = append(batch.latestFloat, StoreLatestFloat{
		Value: value,
		Sub:   sub,
	})
}

func (batch *StoreBatch) Save(write tsdb.DbProtocolWrite) error {
	model := batch.model
	now := time.Now().Unix()

	// Points in time - InfluxDB

	if len(batch.points) != 0 {
		bps := batch.points
		batch.points = nil

		err := write.Write(bps)

		if err != nil {
			return err
		}
	}

	// Text, latest - Mongo

	if len(batch.texts) != 0 {
		for _, item := range batch.texts {

			filter := bson.D{
				{Key: "$and",
					Value: bson.A{
						bson.M{"account_id": batch.account},
						bson.M{"node_id": batch.node},
						bson.M{"item": item.Item},
						bson.M{"sub": item.Sub},
					},
				},
			}

			update := bson.M{
				"$set": bson.M{
					"value": item.Value,
					"when":  now,
				},
			}
			upsert := true

			_, err := model.monCollValueText.UpdateOne(context.Background(), filter, update,
				&options.UpdateOptions{Upsert: &upsert})
			if err != nil {
				fmt.Printf("Mongo error: %s\n", err)
			}
		}
	}

	// UInt value, latest - Mongo

	if len(batch.latestUInt) != 0 {
		for _, item := range batch.latestUInt {
			filter := bson.D{
				{Key: "$and",
					Value: bson.A{
						bson.M{"account_id": batch.account},
						bson.M{"node_id": batch.node},
						bson.M{"sub": item.Sub},
					},
				},
			}

			update := bson.M{
				"$set": bson.M{
					"value_uint": item.Value,
					"when":       now,
				},
			}
			upsert := true

			_, err := model.monCollValueNumeric.UpdateOne(context.Background(), filter, update,
				&options.UpdateOptions{Upsert: &upsert})
			if err != nil {
				fmt.Printf("Mongo error: %s\n", err)
			}
		}
	}

	// Float value, latest - Mongo

	if len(batch.latestFloat) != 0 {
		for _, item := range batch.latestFloat {
			filter := bson.D{
				{Key: "$and",
					Value: bson.A{
						bson.M{"account_id": batch.account},
						bson.M{"node_id": batch.node},
						bson.M{"sub": item.Sub},
					},
				},
			}

			update := bson.M{
				"$set": bson.M{
					"value_float": item.Value,
					"when":        now,
				},
			}
			upsert := true

			_, err := model.monCollValueNumeric.UpdateOne(context.Background(), filter, update,
				&options.UpdateOptions{Upsert: &upsert})
			if err != nil {
				fmt.Printf("Mongo error: %s\n", err)
			}
		}
	}

	return nil
}

func (batch *StoreBatch) roundValue(value float64) float64 {
	return math.Round(value*100.0) / 100.0
}

func (model *ApiServerModel) QueryDataGet(tech string, rq RqCommonModel, series []string, item string) []RsApiV1GetSeries {

	list := make([]RsApiV1GetSeries, 0)

	fmt.Printf("Query data from %s to %s, point duration = %d\n", rq.StartTime.Format(time.RFC3339),
		rq.EndTime.Format(time.RFC3339), rq.PointDuration)

	for _, s := range series {
		list = model.queryDataGetImpl(list, tech, rq, s, item)
	}

	return list
}

func (model *ApiServerModel) QueryDataDiskOverview(tech string, rq RqCommonModel) []*RsApiV1DiskOverviewItem {

	listDisk := make([]*RsApiV1DiskOverviewItem, 0)
	mapDisk := make(map[string]*RsApiV1DiskOverviewItem)

	if STORE_LISTS_IN_MONGO {
		//  Get overall list from Mongo
		filter := bson.D{
			{Key: "node_id", Value: rq.NodeId},
		}

		projection := bson.M{
			"_id": 1, "name": 1, "when": 1,
			"read_count": 1, "write_count": 1,
			"free_bytes": 1, "total_bytes": 1,
			"free_inodes": 1, "total_inodes": 1,
		}

		cursor, err := model.monCollDisk.Find(context.Background(),
			filter, &options.FindOptions{Projection: projection})
		if err != nil {
			return listDisk
		}

		// Load
		for cursor.Next(context.Background()) {
			d := &LoadDisk{}
			if cursor.Decode(d) == nil {
				listDisk = append(listDisk, &RsApiV1DiskOverviewItem{
					Name:       d.Name,
					ReadOps:    d.ReadCount,
					WriteOps:   d.WriteCount,
					SpaceFree:  d.FreeBytes,
					SpaceTotal: d.TotalBytes,
					INodeFree:  d.FreeINodes,
					INodeTotal: d.TotalINodes,
				})
			}
		}

		// Close
		_ = cursor.Close(context.Background())
	} else if tech == defs.TECH_INFLUX {
		// Influx DB - API library
		fmt.Printf("Query data from %s to %s, point duration = %d\n",
			rq.StartTime.Format(time.RFC3339), rq.EndTime.Format(time.RFC3339), rq.PointDuration)

		c := fmt.Sprintf("SELECT MEAN(value), time FROM %s WHERE node_id = '%s' "+
			"AND time >= '%s' AND time < '%s' GROUP BY item, sub",
			"disk_list", rq.NodeId,
			rq.StartTime.Format(time.RFC3339), rq.EndTime.Format(time.RFC3339))

		q := influx.Query{
			Command:  c,
			Database: model.dbNameInflux,
		}

		fmt.Printf("query: %s\n", q.Command)

		if response, err := model.influxRead.Query(q); err == nil && response.Error() == nil {
			fmt.Printf("len response.Results = %d\n", len(response.Results))
			for _, r := range response.Results {
				fmt.Printf("len r.Series = %d\n", len(r.Series))

				for _, s := range r.Series {
					fmt.Printf("len s.Values = %d, name = %s, cols = %s, tags = %s\n", len(s.Values),
						s.Name, s.Columns, s.Tags)

					sItem := s.Tags["item"]
					sSub := s.Tags["sub"]

					if len(sItem) > 0 && len(sSub) > 0 && len(s.Values) == 1 {

						disk, ok := mapDisk[sItem]
						if !ok {
							disk = &RsApiV1DiskOverviewItem{Name: sItem}
							listDisk = append(listDisk, disk)
							mapDisk[sItem] = disk
						}

						v := s.Values[0]
						if v[1] != nil {
							w, err := strconv.ParseFloat(string(v[1].(json.Number)), 64)
							if err == nil {
								w = math.Round(float64(w)*10000.0) / 10000.0
							}
							switch sSub {
							case "read_count":
								disk.ReadOps = uint64(w)
							case "write_count":
								disk.WriteOps = uint64(w)
							case "total_bytes":
								disk.SpaceTotal = uint64(w)
							case "free_bytes":
								disk.SpaceFree = uint64(w)
							case "total_inodes":
								disk.INodeTotal = uint64(w)
							case "free_inodes":
								disk.INodeFree = uint64(w)
							}
						}
					}
				}
			}
		}
	}

	return listDisk
}

func (model *ApiServerModel) QueryDataProcessOverview(tech string, rq RqCommonModel) []*RsApiV1ProcessOverviewItem {

	listProcess := make([]*RsApiV1ProcessOverviewItem, 0)
	mapProcess := make(map[string]*RsApiV1ProcessOverviewItem)

	if STORE_LISTS_IN_MONGO {
		//  Get overall list from Mongo
		filter := bson.D{
			{Key: "node_id", Value: rq.NodeId},
		}

		projection := bson.M{
			"_id": 1, "name": 1, "when": 1,
			"instance_count": 1, "rss": 1, "cpu": 1,
			"chars_read": 1, "chars_write": 1,
			"bytes_read": 1, "bytes_write": 1,
		}

		collection := model.monCollProcess
		cursor, err := collection.Find(context.Background(),
			filter, &options.FindOptions{Projection: projection})
		if err != nil {
			return listProcess
		}

		// Load
		for cursor.Next(context.Background()) {
			p := &LoadProcess{}
			if cursor.Decode(p) == nil {
				i := strings.IndexByte(p.Name, '|')
				if i > 0 {
					sItemName := p.Name[:i]
					sItemUser := p.Name[i+1:]

					listProcess = append(listProcess, &RsApiV1ProcessOverviewItem{
						Name:         sItemName,
						User:         sItemUser,
						Count:        p.InstanceCount,
						Memory:       p.Rss,
						IOCharsRead:  p.CharsRead,
						IOCharsWrite: p.CharsWrite,
						IOBytesRead:  p.BytesRead,
						IOBytesWrite: p.BytesWrite,
						CPU:          p.Cpu,
					})
				}
			}
		}

		// Close
		_ = cursor.Close(context.Background())
	} else if tech == defs.TECH_INFLUX {
		// Influx DB - API library
		fmt.Printf("Query data from %s to %s, point duration = %d\n",
			rq.StartTime.Format(time.RFC3339), rq.EndTime.Format(time.RFC3339), rq.PointDuration)

		c := fmt.Sprintf("SELECT MEAN(value), time FROM %s WHERE node_id = '%s' "+
			"AND time >= '%s' AND time < '%s' GROUP BY item, sub",
			"process_list", rq.NodeId, rq.StartTime.Format(time.RFC3339), rq.EndTime.Format(time.RFC3339))

		q := influx.Query{
			Command:  c,
			Database: model.dbNameInflux,
		}

		fmt.Printf("query: %s\n", q.Command)

		if response, err := model.influxRead.Query(q); err == nil && response.Error() == nil {
			fmt.Printf("len response.Results = %d\n", len(response.Results))
			for _, r := range response.Results {
				fmt.Printf("len r.Series = %d\n", len(r.Series))

				for _, s := range r.Series {
					fmt.Printf("len s.Values = %d, name = %s, cols = %s, tags = %s\n", len(s.Values),
						s.Name, s.Columns, s.Tags)

					sItem := s.Tags["item"]
					sSub := s.Tags["sub"]

					if len(sItem) > 0 && len(sSub) > 0 && len(s.Values) == 1 {
						i := strings.IndexByte(sItem, '|')
						if i > 0 {
							sItemName := sItem[:i]
							sItemUser := sItem[i+1:]

							process, ok := mapProcess[sItem]
							if !ok {
								process = &RsApiV1ProcessOverviewItem{Name: sItemName, User: sItemUser}
								listProcess = append(listProcess, process)
								mapProcess[sItem] = process
							}

							v := s.Values[0]
							if v[1] != nil {
								w, err := strconv.ParseFloat(string(v[1].(json.Number)), 64)
								if err == nil {
									w = math.Round(float64(w)*10000.0) / 10000.0
								}
								switch sSub {
								case "rss":
									process.Memory = uint64(w)
								case "instance_count":
									process.Count = uint64(w)
								case "io_chars_read":
									process.IOCharsRead = uint64(w)
								case "io_chars_write":
									process.IOCharsWrite = uint64(w)
								case "io_bytes_read":
									process.IOBytesRead = uint64(w)
								case "io_bytes_write":
									process.IOBytesWrite = uint64(w)
								case "cpu":
									process.CPU = w
								}
							}
						}
					}
				}
			}
		}
	}

	for _, p := range listProcess {
		p.IOTotal = p.IOCharsRead + p.IOCharsWrite + p.IOBytesRead + p.IOBytesWrite
	}

	return listProcess
}

type SyncPortListen struct {
	Id   primitive.ObjectID `bson:"_id" json:"-"`
	Node string             `bson:"node_id" json:"-"`

	User    string `bson:"user" json:"user"`
	Name    string `bson:"name" json:"name"`
	Type    string `bson:"type" json:"type"`
	SrcAddr net.IP `bson:"addr" json:"src_addr"`
	SrcPort uint16 `bson:"port" json:"src_port"`
}

func (item *SyncPortListen) Key() string {
	return fmt.Sprintf("%s.%s-%s-%s:%d",
		item.User, item.Name, item.Type, item.SrcAddr.String(), item.SrcPort)
}

type SyncPortActive struct {
	Id   primitive.ObjectID `bson:"_id" json:"-"`
	Node string             `bson:"node_id" json:"-"`

	User  string `bson:"user" json:"user"`
	Name  string `bson:"name" json:"name"`
	Count uint64 `bson:"count" json:"count"`
}

func (item *SyncPortActive) Key() string {
	return fmt.Sprintf("%s.%s",
		item.User, item.Name)
}

func (model *ApiServerModel) StorePortsListen(node string, listen []*common.DataPortsListenItem) {
	// 1 - load from database what is there
	storedMap := make(map[string]*SyncPortListen)
	_ = model.queryPortsListenImpl(node, func(s *SyncPortListen) {
		storedMap[s.Key()] = s
	})

	// Go through what is "new" data and sync up
	insertList := make([]interface{}, 0)
	deleteList := make([]interface{}, 0)

	for _, item := range listen {
		sync := &SyncPortListen{
			Node: node,
			User: item.User, Name: item.Name,
			Type:    item.Type,
			SrcAddr: item.SrcAddr, SrcPort: item.SrcPort,
		}
		key := sync.Key()
		if stored, ok := storedMap[key]; ok {
			// Exists in database, leave alone
			stored.Id = primitive.NilObjectID
		} else {
			// Does not exist, insert
			sync.Id = primitive.NewObjectID()
			insertList = append(insertList, sync)
		}
	}

	for _, item := range storedMap {
		if !item.Id.IsZero() {
			deleteList = append(deleteList,
				bson.M{"_id": item.Id})
		}
	}

	// We now know what to insert and what to delete, do it
	if len(deleteList) > 0 {
		_, _ = model.monCollPortListen.DeleteMany(context.Background(), bson.D{
			{Key: "$or",
				Value: deleteList,
			},
		})
	}

	if len(insertList) > 0 {
		_, _ = model.monCollPortListen.InsertMany(context.Background(), insertList, &options.InsertManyOptions{})
	}
}

func (model *ApiServerModel) StorePortsActive(node string, active []*common.DataPortsActiveItem) {
	// 1 - load from database what is there
	storedMap := make(map[string]*SyncPortActive)

	_ = model.queryPortsActiveImpl(node, func(s *SyncPortActive) {
		storedMap[s.Key()] = s
	})

	// Go through what is "new" data and sync up
	insertList := make([]interface{}, 0)
	deleteList := make([]interface{}, 0)

	for _, item := range active {
		sync := &SyncPortActive{
			Node: node,
			User: item.User, Name: item.Name,
			Count: item.Count,
		}
		key := sync.Key()
		if stored, ok := storedMap[key]; ok {
			// Exists in database, just update
			if stored.Count != sync.Count {
				_, _ = model.monCollPortActive.UpdateOne(context.Background(),
					bson.M{"_id": stored.Id},
					bson.D{
						{Key: "$set",
							Value: bson.M{
								"count": sync.Count,
							}}},
					&options.UpdateOptions{})
			}
			stored.Id = primitive.NilObjectID
		} else {
			// Does not exist, insert
			sync.Id = primitive.NewObjectID()
			insertList = append(insertList, sync)
		}
	}

	for _, item := range storedMap {
		if !item.Id.IsZero() {
			deleteList = append(deleteList,
				bson.M{"_id": item.Id})
		}
	}

	// We now know what to insert and what to delete, do it
	if len(deleteList) > 0 {
		_, _ = model.monCollPortActive.DeleteMany(context.Background(), bson.D{
			{Key: "$or",
				Value: deleteList,
			},
		})
	}

	if len(insertList) > 0 {
		_, _ = model.monCollPortActive.InsertMany(context.Background(), insertList, &options.InsertManyOptions{})
	}
}

func (model *ApiServerModel) QueryPorts(node string) ([]*SyncPortListen, []*SyncPortActive) {
	// Listen
	listListen := make([]*SyncPortListen, 0)

	_ = model.queryPortsListenImpl(node, func(s *SyncPortListen) {
		listListen = append(listListen, s)
	})

	// Active
	listActive := make([]*SyncPortActive, 0)

	_ = model.queryPortsActiveImpl(node, func(s *SyncPortActive) {
		listActive = append(listActive, s)
	})

	// Done
	return listListen, listActive
}

func (model *ApiServerModel) QuerySystemText(node string) *common.DataSystemText {
	systemInfo := &common.DataSystemText{}

	model.queryTextPropertyImpl(node, func(sub string, value string) {
		switch sub {
		case "cpu":
			systemInfo.CpuLabel = value
		case "kernel":
			systemInfo.KernelLabel = value
		case "os_name":
			systemInfo.OsDistLabel = value
		case "os_version":
			systemInfo.OsVersionLabel = value
		case "app_apache_version":
			systemInfo.AppApacheVersion = value
		case "app_nginx_version":
			systemInfo.AppNginxVersion = value
		case "app_mysql_version":
			systemInfo.AppMysqlVersion = value
		case "app_pgsql_version":
			systemInfo.AppPgsqlVersion = value
		}
	})

	return systemInfo
}

func (model *ApiServerModel) QueryApacheVersion(node string) string {
	version := ""

	model.queryTextPropertyImpl(node, func(sub string, value string) {
		switch sub {
		case "app_apache_version":
			version = value
		}
	})

	return version
}

func (model *ApiServerModel) QueryNginxVersion(node string) string {
	version := ""

	model.queryTextPropertyImpl(node, func(sub string, value string) {
		switch sub {
		case "app_nginx_version":
			version = value
		}
	})

	return version
}

func (model *ApiServerModel) QueryMysqlVersion(node string) string {
	version := ""

	model.queryTextPropertyImpl(node, func(sub string, value string) {
		switch sub {
		case "app_mysql_version":
			version = value
		}
	})

	return version
}

func (model *ApiServerModel) QueryPgsqlVersion(node string) string {
	version := ""

	model.queryTextPropertyImpl(node, func(sub string, value string) {
		switch sub {
		case "app_pgsql_version":
			version = value
		}
	})

	return version
}

func (model *ApiServerModel) StoreDiskListInMongo(nodeId string, diskList []*common.DataDiskItem) error {
	filter := bson.D{
		{Key: "node_id", Value: nodeId},
	}

	projection := bson.M{
		"_id": 1, "name": 1, "when": 1,
		"read_count": 1, "write_count": 1,
		"free_bytes": 1, "total_bytes": 1,
		"free_inodes": 1, "total_inodes": 1,
	}

	collection := model.monCollDisk
	cursor, err := collection.Find(context.Background(),
		filter, &options.FindOptions{Projection: projection})
	if err != nil {
		return err
	}

	defer func() {
		_ = cursor.Close(context.Background())
	}()

	// Load
	storedMap := make(map[string]*LoadDisk)
	for cursor.Next(context.Background()) {
		d := &LoadDisk{}
		if cursor.Decode(d) == nil {
			storedMap[d.Name] = d
		}
	}

	// Diff and update
	now := time.Now().Unix()
	for _, d := range diskList {
		valuesFull := bson.M{
			"when":       now,
			"read_count": d.ReadCountInstant, "write_count": d.WriteCountInstant,
			"free_bytes": d.BytesFree, "total_bytes": d.BytesTotal,
			"free_inodes": d.INodeFree, "total_inodes": d.INodeTotal,
		}
		valuesWhen := bson.M{
			"when": now,
		}
		existing, ok := storedMap[d.Name]
		if ok {
			// Reconcile with existing
			updateById := bson.D{
				{Key: "_id", Value: existing.Id},
			}
			if !existing.Equals(d) {
				_, err = collection.UpdateOne(context.Background(),
					updateById, bson.M{"$set": valuesFull}, &options.UpdateOptions{})
				if err != nil {
					return err
				}
			} else if now-existing.When >= MONGO_LIST_TIME_UPDATE {
				_, err = collection.UpdateOne(context.Background(),
					updateById, bson.M{"$set": valuesWhen}, &options.UpdateOptions{})
			}
			delete(storedMap, d.Name)
		} else {
			// Insert new
			valuesFull["node_id"] = nodeId
			valuesFull["name"] = d.Name

			_, err = collection.InsertOne(context.Background(),
				valuesFull, &options.InsertOneOptions{})
			if err != nil {
				return err
			}
		}
	}

	// Delete any we don't need
	for _, d := range storedMap {
		if now-d.When >= MONGO_LIST_TIME_EXPIRE {
			_, err = collection.DeleteOne(context.Background(),
				bson.D{
					{Key: "_id", Value: d.Id},
				}, &options.DeleteOptions{})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (model *ApiServerModel) StoreProcessListInMongo(nodeId string, processList []*common.DataProcess) error {
	filter := bson.D{
		{Key: "node_id", Value: nodeId},
	}

	projection := bson.M{
		"_id": 1, "name": 1, "when": 1,
		"instance_count": 1, "rss": 1, "cpu": 1,
		"chars_read": 1, "chars_write": 1,
		"bytes_read": 1, "bytes_write": 1,
	}

	collection := model.monCollProcess
	cursor, err := collection.Find(context.Background(),
		filter, &options.FindOptions{Projection: projection})
	if err != nil {
		return err
	}
	defer func() {
		_ = cursor.Close(context.Background())
	}()

	// Load
	storedMap := make(map[string]*LoadProcess)
	for cursor.Next(context.Background()) {
		p := &LoadProcess{}
		if cursor.Decode(p) == nil {
			storedMap[p.Name] = p
		}
	}

	// Diff and update
	now := time.Now().Unix()
	for _, p := range processList {
		valuesFull := bson.M{
			"when":           now,
			"instance_count": p.Count, "rss": p.RSS, "cpu": p.CPUScaled,
			"chars_read": p.IOReadCharsInstant, "chars_write": p.IOWriteCharsInstant,
			"bytes_read": p.IOReadBytesInstant, "bytes_write": p.IOWriteBytesInstant,
		}
		valuesWhen := bson.M{
			"when": now,
		}
		nameAndUser := p.Name + "|" + p.User
		existing, ok := storedMap[nameAndUser]
		if ok {
			// Reconcile with existing
			updateById := bson.D{
				{Key: "_id", Value: existing.Id},
			}
			if !existing.Equals(p, nameAndUser) {
				_, err = collection.UpdateOne(context.Background(),
					updateById, bson.M{"$set": valuesFull}, &options.UpdateOptions{})
				if err != nil {
					return err
				}
			} else if now-existing.When >= MONGO_LIST_TIME_UPDATE {
				_, err = collection.UpdateOne(context.Background(),
					updateById, bson.M{"$set": valuesWhen}, &options.UpdateOptions{})
			}
			delete(storedMap, p.Name)
		} else {
			// Insert new
			valuesFull["node_id"] = nodeId
			valuesFull["name"] = nameAndUser

			_, err = collection.InsertOne(context.Background(),
				valuesFull, &options.InsertOneOptions{})
			if err != nil {
				return err
			}
		}
	}

	// Delete any we don't need
	for _, p := range storedMap {
		if now-p.When >= MONGO_LIST_TIME_EXPIRE {
			_, err = collection.DeleteOne(context.Background(),
				bson.D{
					{Key: "_id", Value: p.Id},
				}, &options.DeleteOptions{})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (model *ApiServerModel) queryDataGetImpl(list []RsApiV1GetSeries, tech string, rq RqCommonModel,
	series string, item string) []RsApiV1GetSeries {

	if tech == defs.TECH_INFLUX {
		// Influx DB - API library
		f := fmt.Sprintf("SELECT MEAN(value), time FROM %s", series)
		c := ""
		if len(item) >= 0 {
			c = fmt.Sprintf("%s WHERE node_id = '%s' AND item = '%s' "+
				"AND time >= '%s' AND time < '%s' GROUP BY sub, time(%ds)",
				f, rq.NodeId, item, rq.StartTime.Format(time.RFC3339), rq.EndTime.Format(time.RFC3339),
				rq.PointDuration)
		} else {
			c = fmt.Sprintf("%s WHERE node_id = '%s' "+
				"AND time >= '%s' AND time < '%s' GROUP BY sub, time(%ds)",
				f, rq.NodeId, rq.StartTime.Format(time.RFC3339), rq.EndTime.Format(time.RFC3339),
				rq.PointDuration)
		}

		q := influx.Query{
			Command:  c,
			Database: model.dbNameInflux,
		}

		fmt.Printf("query: %s\n", q.Command)

		if response, err := model.influxRead.Query(q); err == nil && response.Error() == nil {
			fmt.Printf("len response.Results = %d\n", len(response.Results))
			for _, r := range response.Results {
				fmt.Printf("len r.Series = %d\n", len(r.Series))

				for _, s := range r.Series {
					fmt.Printf("len s.Values = %d, name = %s:%s, cols = %s, tags = %s\n",
						len(s.Values), series, s.Name, s.Columns, s.Tags)

					sSub := s.Tags["sub"]
					listPoints := make([]RsApiV1GetPoint, 0)

					for _, v := range s.Values {
						if len(v) >= 2 {
							// Time, Value
							when := int64(-1)
							value := float64(-1.0)
							isValueNil := false

							//fmt.Printf("%s %s\n", reflect.TypeOf(v[0]), reflect.TypeOf(v[1]))
							w, err := time.Parse("2006-01-02T15:04:05Z", v[0].(string))
							if err == nil {
								when = w.Unix()
							}
							if v[1] == nil {
								isValueNil = true
							} else {
								w, err := strconv.ParseFloat(string(v[1].(json.Number)), 64)
								if err == nil {
									value = math.Round(float64(w)*10000.0) / 10000.0
								}
							}

							listPoints = append(listPoints, RsApiV1GetPoint{PointTime: when,
								PointValue: value,
								IsValueNil: isValueNil})
						}
					}

					list = append(list, RsApiV1GetSeries{
						SubValue: series + ":" + sSub,
						Points:   listPoints})
				}
			}
		}
	}

	return list
}

func (model *ApiServerModel) queryTextPropertyImpl(node string, call func(sub string, value string)) {
	filter := bson.D{
		{Key: "$and",
			Value: bson.A{
				bson.M{"node_id": node},
				bson.M{"item": ""},
			},
		},
	}

	projection := bson.M{
		"item":  1,
		"sub":   1,
		"value": 1,
	}

	cursor, err := model.monCollValueText.Find(context.Background(),
		filter, &options.FindOptions{Projection: projection})
	if err == nil {
		for cursor.Next(context.Background()) {
			var s LoadText
			if cursor.Decode(&s) == nil {
				call(s.Sub, s.Value)
			}
		}
		_ = cursor.Close(context.Background())
	}
}

func (model *ApiServerModel) queryPortsListenImpl(node string, call func(s *SyncPortListen)) error {
	filter := bson.M{
		"node_id": node,
	}

	projection := bson.M{
		"_id": 1, "node_id": 1,
		"user": 1, "name": 1,
		"type": 1,
		"addr": 1, "port": 1,
	}

	cursor, err := model.monCollPortListen.Find(context.Background(),
		filter, &options.FindOptions{Projection: projection})
	if err != nil {
		return err
	}

	defer func() {
		_ = cursor.Close(context.Background())
	}()

	for cursor.Next(context.Background()) {
		var s *SyncPortListen
		if cursor.Decode(&s) == nil {
			call(s)
		}
	}

	return nil
}

func (model *ApiServerModel) queryPortsActiveImpl(node string, call func(s *SyncPortActive)) error {
	filter := bson.M{
		"node_id": node,
	}

	projection := bson.M{
		"_id": 1, "node_id": 1,
		"user": 1, "name": 1,
		"count": 1,
	}

	cursor, err := model.monCollPortActive.Find(context.Background(),
		filter, &options.FindOptions{Projection: projection})
	if err != nil {
		return err
	}

	defer func() {
		_ = cursor.Close(context.Background())
	}()

	for cursor.Next(context.Background()) {
		var s *SyncPortActive
		if cursor.Decode(&s) == nil {
			call(s)
		}
	}

	return nil
}

func (model *ApiServerModel) QueryLatestForAll() []*RsApiV1NodeNumeric {
	now := time.Now().Unix()
	m := make(map[string]*RsApiV1NodeNumeric)

	// 1 - Load nodes with id's and names

	filterNode := bson.D{}

	projectionNode := bson.M{
		"node_id":    1,
		"node_title": 1,
	}

	cursor, err := model.monCollNode.Find(context.Background(),
		filterNode, &options.FindOptions{Projection: projectionNode})
	if err == nil {
		for cursor.Next(context.Background()) {
			var s LoadNode
			if cursor.Decode(&s) == nil {
				l := &RsApiV1NodeNumeric{
					Node:  s.Node,
					Title: s.Title,
				}
				m[l.Node] = l
			}
		}
		_ = cursor.Close(context.Background())
	}

	// 2 - Load data and cross-link to nodes

	filterData := bson.D{
		{
			Key: "when",
			Value: bson.M{
				"$gte": now - 60*60,
			},
		},
	}

	projectionData := bson.M{
		"node_id":     1,
		"sub":         1,
		"when":        1,
		"value_uint":  1,
		"value_float": 1,
	}

	cursor, err = model.monCollValueNumeric.Find(context.Background(),
		filterData, &options.FindOptions{Projection: projectionData})
	if err == nil {
		memory := common.DataSystemMemory{}
		for cursor.Next(context.Background()) {
			var item LoadLatestNumericItem
			if cursor.Decode(&item) == nil {
				if node, ok := m[item.Node]; ok {
					model.decodeLatestValuesImpl(&item, node, &memory)
				}
			}
		}
		_ = cursor.Close(context.Background())
	}

	l := make([]*RsApiV1NodeNumeric, 0)
	for _, v := range m {
		l = append(l, v)
	}

	return l
}

func (model *ApiServerModel) QuerySystemNumericAndMemory(node string,
	numeric *RsApiV1NodeNumeric, memory *common.DataSystemMemory) {
	now := time.Now().Unix()

	// 1 - Load data for node

	filterData := bson.D{
		{Key: "$and",
			Value: bson.A{
				bson.M{"node_id": node},
				bson.M{"when": bson.M{
					"$gte": now - 60*60,
				}},
			},
		},
	}

	projectionData := bson.M{
		"account_id":  1,
		"node_id":     1,
		"sub":         1,
		"when":        1,
		"value_uint":  1,
		"value_float": 1,
	}

	cursor, err := model.monCollValueNumeric.Find(context.Background(),
		filterData, &options.FindOptions{Projection: projectionData})
	if err == nil {
		for cursor.Next(context.Background()) {
			var item LoadLatestNumericItem
			if cursor.Decode(&item) == nil {
				model.decodeLatestValuesImpl(&item, numeric, memory)
			}
		}
		_ = cursor.Close(context.Background())
	}
}

func (model *ApiServerModel) decodeLatestValuesImpl(item *LoadLatestNumericItem,
	numeric *RsApiV1NodeNumeric, memory *common.DataSystemMemory) {
	switch item.Sub {
	// numeric
	case "cpu":
		numeric.ValueCpu = item.ValueFloat
	case "mem":
		numeric.ValueMemory = item.ValueUInt
	case "swap":
		numeric.ValueSwap = item.ValueUInt
	case "load":
		numeric.ValueLoad = item.ValueFloat
	case "net":
		numeric.ValueNetwork = item.ValueUInt
	case "cpun":
		numeric.ValueCpuN = item.ValueUInt
	// memory
	case "realmemsize":
		memory.RealMemorySize = item.ValueUInt
	case "realmemused":
		memory.RealMemoryUsed = item.ValueUInt
	case "swapsize":
		memory.SwapMemorySize = item.ValueUInt
	case "swapused":
		memory.SwapMemoryUsed = item.ValueUInt
	case "disksize":
		memory.DiskTotalSize = item.ValueUInt
	case "diskused":
		memory.DiskTotalUsed = item.ValueUInt
	default:
		fmt.Printf("Unknown sub: %s\n", item.Sub)
	}

	if numeric.When < item.When {
		numeric.When = item.When
	}
}

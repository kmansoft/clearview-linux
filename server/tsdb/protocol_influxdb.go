package tsdb

import (
	influx "github.com/orourkedd/influxdb1-client"
	"net/url"
)

type DbProtocolWriteInflux struct {
	client       *influx.Client
	databaseName string
}

func NewDbProtocolWriteInflux(influxUrl url.URL, databaseName string) (*DbProtocolWriteInflux, error) {
	config := influx.Config{
		URL: influxUrl,
	}
	client, err := influx.NewClient(config)
	if err != nil {
		return nil, err
	}

	return &DbProtocolWriteInflux{
		client:       client,
		databaseName: databaseName,
	}, nil
}

func (write *DbProtocolWriteInflux) Write(points []influx.Point) error {
	batch := influx.BatchPoints{
		Points:   points,
		Database: write.databaseName}

	_, err := write.client.Write(batch)
	return err
}

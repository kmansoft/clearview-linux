package tsdb

import (
	influx "github.com/orourkedd/influxdb1-client"
)

type DbProtocolWrite interface {
	Write(points []influx.Point) error
}

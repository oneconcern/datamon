package influxdb

import (
	"context"
	"time"

	influxdb "github.com/influxdata/influxdb/client/v2"
)

// MetricPoint represents a single row in a batch of measurements
type MetricPoint struct {
	Measurement string
	Tags        map[string]string
	Fields      map[string]interface{}
	Timestamp   time.Time
}

// Store provides an access to an influxdb database for reading and writing metrics
type Store interface {
	Database() string
	GetClient() influxdb.Client
	Ping(context.Context, time.Duration) error
	ReadMetrics(context.Context, string) (*influxdb.Response, error)
	WriteMetrics(context.Context, string, map[string]string, map[string]interface{}) error
	WriteBatch(context.Context, []MetricPoint) error
}

var _ Store = &influxDB{}

type influxDB struct {
	config   influxdb.HTTPConfig
	client   influxdb.Client
	database string
	mapper   func(string, map[string]string) (string, map[string]string)
}

func defaultInfluxDB() *influxDB {
	return &influxDB{
		config: influxdb.HTTPConfig{
			Addr:               "http://localhost:8086",
			Username:           "admin",
			InsecureSkipVerify: true,
		},
		database: "test",
	}
}

// NewStore builds an instance of Store with some options
func NewStore(opts ...StoreOption) (Store, error) {
	db := defaultInfluxDB()
	for _, apply := range opts {
		apply(db)
	}
	c, err := influxdb.NewHTTPClient(db.config)
	if err != nil {
		return nil, err
	}
	db.client = c
	return db, nil
}

func (db *influxDB) GetClient() influxdb.Client {
	return db.client
}

func (db *influxDB) Database() string {
	return db.database
}

func (db *influxDB) Ping(_ context.Context, timeout time.Duration) error {
	_, _, err := db.client.Ping(timeout)
	if err != nil {
		return err
	}
	return nil
}

func (db *influxDB) WriteMetrics(_ context.Context, measurement string, tags map[string]string, fields map[string]interface{}) error {
	bp, err := influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
		Database:  db.database,
		Precision: "s",
	})

	if err != nil {
		return err
	}

	if db.mapper != nil {
		measurement, tags = db.mapper(measurement, tags)
	}

	point, err1 := influxdb.NewPoint(
		measurement,
		tags,
		fields,
		time.Now(),
	)
	if err1 != nil {
		return err1
	}

	bp.AddPoint(point)

	err = db.client.Write(bp)
	if err != nil {
		return err
	}
	return nil
}

func (db *influxDB) ReadMetrics(_ context.Context, query string) (*influxdb.Response, error) {
	q := influxdb.Query{
		Command:  query,
		Database: db.database,
	}
	resp, err := db.client.Query(q)
	if err != nil {
		return nil, err
	}
	if resp.Error() != nil {
		return nil, err
	}
	return resp, nil
}

func (db *influxDB) WriteBatch(_ context.Context, points []MetricPoint) error {
	bp, err := influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
		Database:  db.database,
		Precision: "s",
	})
	if err != nil {
		return err
	}
	for _, point := range points {
		if db.mapper != nil {
			point.Measurement, point.Tags = db.mapper(point.Measurement, point.Tags)
		}

		pt, erp := influxdb.NewPoint(point.Measurement, point.Tags, point.Fields, point.Timestamp)
		if erp != nil {
			return erp
		}
		bp.AddPoint(pt)

	}
	err = db.client.Write(bp)
	if err != nil {
		return err
	}
	return nil
}

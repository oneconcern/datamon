package metrics

import (
	"context"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/oneconcern/datamon/pkg/metrics/exporters/influxdb"

	"github.com/docker/go-units"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

const (
	// KB stands for kilo bytes (1024 bytes)
	KB = units.KiB

	// MB stands for mega bytes (1024 kilo bytes)
	MB = units.MiB

	// GB stands for giga bytes (1024 mega bytes)
	GB = units.GiB

	unitCount    = "count"
	unitSumBytes = "sumbytes"
	unitBps      = "bps"
)

var (
	// global settings for metrics
	mp       *settings
	initOnce sync.Once
)

type settings struct {
	basePath  string
	contexter func() context.Context
	exporter  view.Exporter

	allMetrics []stats.Measure
	allViews   []*view.View

	// a map of all registered modules
	modules   map[string]interface{}
	exclusive sync.Mutex

	d time.Duration
}

// defaultSettings define an exporter to a local influxDB backend store
func defaultSettings() *settings {
	return &settings{
		modules:   make(map[string]interface{}),
		contexter: context.Background,
		// default reporting period is left to the default from opencensus exporter (10s)
	}
}

func defaultStore() influxdb.Store {
	sink, _ := influxdb.NewStore(
		influxdb.WithDatabase("datamon"),
		influxdb.WithNameAsTag("metrics"), // use metric name as an influxdb tag, with unique time series "metrics"
	)
	return sink
}

// DefaultExporter returns a metrics exporter for an influxdb backend, with db "datamon" and time series "metrics"
func DefaultExporter(opts ...influxdb.Option) view.Exporter {
	return flusher(influxdb.NewExporter(
		append([]influxdb.Option{
			influxdb.WithStore(defaultStore()),
			influxdb.WithTags(map[string]string{"service": "datamon"}),
		}, opts...)...,
	))
}

func newSettings(opts ...Option) *settings {
	s := defaultSettings()
	for _, apply := range opts {
		apply(s)
	}

	if s.exporter == nil {
		s.exporter = DefaultExporter()
	}

	s.RegisterExporter()
	return s
}

func (s *settings) EnsureMetrics(location string, m interface{}) interface{} {
	s.exclusive.Lock()
	defer s.exclusive.Unlock()
	location = path.Join(s.basePath, location)

	if existing, ok := s.modules[location]; ok {
		if !equalType(existing, m) {
			panic("trying to re-register existing metrics module with a different type")
		}
		return existing
	}
	scanStruct(location, s.addMetric, m)
	s.modules[location] = m
	return m
}

// Flush collects all remaining data for registered views and exports them
func (s *settings) Flush() {
	for _, v := range s.allViews {
		rows, err := view.RetrieveData(v.Name)
		if err != nil {
			continue // ignore errors when pushing metrics
		}
		data := &view.Data{
			View:  v,
			Start: time.Now(), // cannot figure out last snapshot time from the background worker
			End:   time.Now(),
			Rows:  rows,
		}
		s.exporter.ExportView(data)
	}
}

// registerExporter registers the current set exporter to the opencensus library
func (s *settings) RegisterExporter() {
	if s.exporter != nil {
		view.RegisterExporter(s.exporter)
		if s.d >= time.Second {
			view.SetReportingPeriod(s.d)
		}
	}
}

// addMetric creates a metric with some views, according to the decoded struct tags.
//
// Every metric is created with a default view according to its unit type:
//   - counters (unit=unitCount or "") get a count view
//   - bytes get a bytes size distribution view
//   - timings get a duration distribution view
//   - throughputs (bps) get a throughput distribution view
//   - sumbytes get a cumulated bytes size sum view
//
// Supported extra views can be defined in struct tags, e.g. views:"sum,lastvalue,count"
func (s *settings) addMetric(m interface{}, metric, group string, tags map[string]string) interface{} {
	name := path.Join(group, metric)
	description := tags["description"]
	unit := tags["unit"]

	if description == "" {
		description = describeFromTags(name, tags)
	}
	// define default view
	u, dist := unitAndDist(unit)

	var measure stats.Measure
	switch m.(type) {
	case *stats.Int64Measure:
		measure = stats.Int64(name, description, u)
	case *stats.Float64Measure:
		measure = stats.Float64(name, description, u)
	default:
		return nil
	}

	s.allMetrics = append(s.allMetrics, measure)

	// capturing tags in views
	groupingTag := tags["groupings"]
	groupings := strings.Split(groupingTag, ",")
	keys := make([]tag.Key, 0, len(groupings))
	for _, g := range groupings {
		if g != "" {
			keys = append(keys, tag.MustNewKey(g))
		}
	}

	viewOnMetric := &view.View{
		Name:        name,
		Description: describeViewFromDist(description, dist),
		Measure:     measure,
		Aggregation: dist,
		TagKeys:     keys,
	}
	s.allViews = append(s.allViews, viewOnMetric)
	_ = view.Register(viewOnMetric)

	extraViews := tags["views"]
	if extraViews != "" {
		// add extra views
		extras := strings.Split(extraViews, ",")
		for _, extra := range extras {
			extraView := &view.View{
				Measure: measure,
				TagKeys: keys,
			}
			switch extra {
			case unitCount:
				extraView.Aggregation = view.Count()
			case "sum":
				extraView.Aggregation = view.Sum()
			case "lastvalue":
				extraView.Aggregation = view.LastValue()
			}
			if extraView.Aggregation != nil {
				extraView.Name = describeViewFromDist(name, extraView.Aggregation)
				extraView.Description = describeViewFromDist(description, extraView.Aggregation)
				s.allViews = append(s.allViews, extraView)
				_ = view.Register(extraView)
			}
		}
	}
	return measure
}

func durationDistribution() *view.Aggregation {
	// buckets in milliseconds
	return view.Distribution(
		10, 50,
		100, 300, 500, 700, 900,
		1000, 1300, 1500, 1700, 1900,
		2000, 3000, 5000, 7000, 9000,
		10000, 30000, 50000, 70000, 90000,
		100000,
	)
}

func bytesDistribution() *view.Aggregation {
	// buckets in bytes
	return view.Distribution(
		500,
		1*KB, 5*KB, 10*KB, 50*KB,
		100*KB, 500*KB, 1*GB,
		1.5*GB, /* cut-off at default cafs leaf size */
		5*GB, 10*GB, 50*GB,
		100*GB, 500*GB, 1000*GB,
	)
}

func throughputDistribution() *view.Aggregation {
	return view.Distribution(
		1*KB, 5*KB, 50*KB, 100*KB, // for small files
		1*MB,
		10*MB,
		20*MB,
		50*MB,
		100*MB,
		150*MB,
	)
}

func unitAndDist(unit string) (string, *view.Aggregation) {
	switch unit {
	case "milliseconds":
		return stats.UnitMilliseconds, durationDistribution()
	case "bytes":
		return stats.UnitBytes, bytesDistribution()
	case unitSumBytes:
		return stats.UnitBytes, view.Sum()
	case "bytespersec", unitBps:
		return unitBps, throughputDistribution()
	case unitCount:
		fallthrough
	default:
		return stats.UnitDimensionless, view.Count()
	}
}

func describeFromTags(name string, tags map[string]string) string {
	unit := tags["unit"]
	switch unit {
	case unitSumBytes:
		name += " cumulated bytes"
	case "", unitCount:
		name += " counter"
	default:
		name += " in " + unit
	}
	return name
}

func describeViewFromDist(desc string, in *view.Aggregation) string {
	if in == nil {
		return desc
	}
	switch in.Type {
	case view.AggTypeCount:
		return desc + " [count]"
	case view.AggTypeSum:
		return desc + " [cumulated]"
	case view.AggTypeDistribution:
		return desc + " [distribution]"
	case view.AggTypeLastValue:
		return desc + " [last]"
	case view.AggTypeNone:
		fallthrough
	default:
		return desc
	}
}

// FlushExporter is a view exporter that knows how to flush metrics.
//
// This basically means that we may export views concurrently with the default
// background exporter.
type FlushExporter interface {
	view.Exporter
	Flush(*view.Data)
}

// flusher makes a FlushExporter of view.Exporter
func flusher(e view.Exporter) FlushExporter {
	return &simpleFlusher{
		e: e,
	}
}

type simpleFlusher struct {
	e view.Exporter
	m sync.RWMutex
}

func (f *simpleFlusher) ExportView(viewData *view.Data) {
	f.m.RLock() // we don't want to lock out the view background worker, which may parallelize things however it sees fit
	f.e.ExportView(viewData)
	f.m.RUnlock()
}

func (f *simpleFlusher) Flush(viewData *view.Data) {
	f.m.Lock()
	f.e.ExportView(viewData)
	f.m.Unlock()
}

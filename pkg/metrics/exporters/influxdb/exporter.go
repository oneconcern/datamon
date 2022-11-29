package influxdb

import (
	"context"
	"fmt"
	"strings"

	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var _ view.Exporter = &Exporter{}

func defaultExporter() *Exporter {
	sink, _ := NewStore()
	return &Exporter{
		errorHandler: func(_ error) {},
		store:        sink,
	}
}

// NewExporter creates a new Influxdb exporter.
//
// Use options to configure:
//   - an influxdd.Store instance, configured with the desired settings
//   - an error handler. If set to nil, a no-op handler is set by default
//   - a map of custom tags for written records (may be nil)
func NewExporter(opts ...Option) *Exporter {
	e := defaultExporter()
	for _, apply := range opts {
		apply(e)
	}
	return e
}

const (
	// tags to represent opencensus information as influxdb tags
	descriptionTag = "description" // view description
	unitTag        = "unit"        // measurement unit
	groupingTag    = "grouping"    // view aggregation/filtering tag
	aggregationTag = "aggregation" // view aggregation type (count, sum, last, distribution)

	// opencensus information represented as inluxdb fields
	startField       = "start"             // start of the view aggregation period
	observationField = "observationPeriod" // duration of the view aggregation period
	valueField       = "value"
	minField         = "min" // statistics on distribution aggregations
	maxField         = "max"
	meanField        = "mean"
	countField       = "count"
	bucketsField     = "buckets" // buckets on distribution aggregations
)

// Exporter is an opencensus exporter for Influxdb
type Exporter struct {
	store        Store
	errorHandler func(error)
	customTags   map[string]string
}

// ExportView sends collected metrics to the backend sink
func (e *Exporter) ExportView(viewData *view.Data) {
	points := make([]MetricPoint, 0, len(viewData.Rows))
	for i, row := range viewData.Rows {
		fields := make(map[string]interface{}, len(viewData.Rows))
		tags := make(map[string]string, len(e.customTags)+len(row.Tags)+2)

		// view metadata
		fields[startField] = viewData.Start
		fields[observationField] = viewData.End.Sub(viewData.Start)
		if viewData.View.Description != "" {
			tags[descriptionTag] = viewData.View.Description
		}
		tags[unitTag] = viewData.View.Measure.Unit()

		// view aggregation keys
		if i < len(viewData.View.TagKeys) {
			tags[groupingTag] = strings.ToLower(viewData.View.TagKeys[i].Name())
		}

		// view fields
		switch d := row.Data.(type) {
		case *view.CountData:
			fields[valueField] = float64(d.Value)
			tags[aggregationTag] = "count"
		case *view.DistributionData:
			fields[minField] = d.Min
			fields[maxField] = d.Max
			fields[meanField] = d.Mean
			fields[countField] = d.Count
			fields[bucketsField] = d.CountPerBucket
			tags[aggregationTag] = "distribution"
		case *view.LastValueData:
			fields[valueField] = d.Value
			tags[aggregationTag] = "last"
		case *view.SumData:
			fields[valueField] = d.Value
			tags[aggregationTag] = "sum"
		default:
			e.errorHandler(fmt.Errorf("unknown AggregationData type: %T", row.Data))
			return
		}

		appendAndReplace(tags, e.customTags)
		appendAndReplace(tags, convertTags(row.Tags))

		points = append(points, MetricPoint{
			Measurement: viewData.View.Name,
			Tags:        tags,
			Fields:      fields,
			Timestamp:   viewData.End,
		})
	}

	if err := e.store.WriteBatch(context.Background(), points); err != nil {
		e.errorHandler(err)
	}
}

// appendAndReplace appends all the data from the 'src' to the
// 'dst' map. If both have the same key, the one from 'src' is taken.
func appendAndReplace(dst, src map[string]string) {
	if dst == nil {
		dst = make(map[string]string, len(src))
	}

	for k, v := range src {
		dst[k] = v
	}
}

func convertTags(tags []tag.Tag) map[string]string {
	res := make(map[string]string)
	for _, tag := range tags {
		res[tag.Key.Name()] = tag.Value
	}
	return res
}

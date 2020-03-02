package metrics

import (
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

// Init global settings for metrics collection, such as global tags and exporter setup.
//
// Init is used by any top-level package (such as CLI driver or SDK), to define global
// settings such as exporter and global tags.
//
// Init may be called multiple times: only the first time matters.
//
// Metrics and views may be registered at init time or later on.
func Init(opts ...Option) {
	initOnce.Do(func() {
		mp = newSettings(opts...)
	})
}

// Flush all collected metrics to backend
func Flush() {
	mp.Flush()
}

// EnsureMetrics allows for lazy registration of metrics definitions.
//
// It may safely be called several times, and only the first registration
// for a given unique location will be retained.
//
// When running several times, it ensures that all subsequent calls on the same location
// specify the same metrics type, otherwise it panics.
func EnsureMetrics(location string, m interface{}) interface{} {
	return mp.EnsureMetrics(location, m)
}

// Inc increments a counter-like metric
func Inc(counter *stats.Int64Measure, tags ...map[string]string) {
	_ = stats.RecordWithTags(mp.contexter(), mergeTags(tags), counter.M(1))
}

// Int64 sets a value to a measurement
func Int64(measure *stats.Int64Measure, value int64, tags ...map[string]string) {
	_ = stats.RecordWithTags(mp.contexter(), mergeTags(tags), measure.M(value))
}

// Float64 sets a value to a measurement
func Float64(measure *stats.Float64Measure, value float64, tags ...map[string]string) {
	_ = stats.RecordWithTags(mp.contexter(), mergeTags(tags), measure.M(value))
}

// Since feeds a millisecs timing measurement from some start time
func Since(start time.Time, measure *stats.Float64Measure, tags ...map[string]string) {
	ms := float64(time.Since(start).Nanoseconds()) / 1e6
	_ = stats.RecordWithTags(mp.contexter(), mergeTags(tags), measure.M(ms))
}

// Duration feeds a millisecs timing measurement from some start to end timings
func Duration(start, end time.Time, measure *stats.Float64Measure, tags ...map[string]string) {
	ms := float64(end.Sub(start).Nanoseconds()) / 1e6
	_ = stats.RecordWithTags(mp.contexter(), mergeTags(tags), measure.M(ms))
}

// mergeTags adds some dynamically defined tags to a single measurement
func mergeTags(extras []map[string]string) []tag.Mutator {
	mutators := make([]tag.Mutator, 0, 10)
	for _, extra := range extras {
		for k, v := range extra {
			mutators = append(mutators, tag.Upsert(tag.MustNewKey(k), v))
		}
	}
	return mutators
}

// Enable equips any type with some capabilities to collect metrics in a very concise way.
//
// Sample usage:
//
//   type myType struct{
//     ...
//     metrics.Enable
//     m *myMetrics // m points to the globally registered metrics collector
//   }
//
//   ...
//
//   // MyTypeUsage describes a tree of metrics to be recorded on myType
//   type MyTypeUsage struct {
//     Volumetry struct {
//       Metadata  FilesMetrics `group:"metadata" description:"some file metrics issued by myType"`
//       TestCount *stats.Int64Measure   `metric:"testCount" description:"number of tests" extraviews:"sum"`
//     } `group:"volumetry" description:"volumetry measurements that pertain to myType"`
//   }
//
//   func (u *MyTypeUsage) Reads() {
//     metrics.Inc(u.Volumetry.Metadata.Read)
//   }
//
//   func (u *MyTypeUsage) Tests(p int) {
//     metrics.Int64(u.Volumetry.TestCount, intt64(p))
//   }
//
//   ...
//
//   func NewMyType() *myType {
//     ...
//     t := &MyType{}
//     t.m := t.EnsureMetrics("MyType", &myMetrics{})
//     t.EnableMetrics(true)
//     return t
//   }
type Enable struct {
	metricsEnabled bool
}

// MetricsEnabled tells whether metrics are enabled or not
func (e Enable) MetricsEnabled() bool {
	return e.metricsEnabled
}

// EnableMetrics toggles metrics collection
func (e *Enable) EnableMetrics(enabled bool) {
	e.metricsEnabled = enabled
}

// EnsureMetrics registers a type describing metrics to the global metrics collection.
// The name argument constructs a new path in the metrics tree.
//
// EnsureMetrics may be called several times, only the first registration will apply.
//
// EnsureMetrics may be called lazily: metrics collection will start only after the first registration.
//
// NOTE: EnsureMetrics will panic if not called with a pointer to a struct.
func (e *Enable) EnsureMetrics(name string, m interface{}) interface{} {
	return EnsureMetrics(name, m)
}

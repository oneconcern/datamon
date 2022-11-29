package metrics

import (
	"time"

	"go.opencensus.io/stats"
)

// FilesMetrics is common set of metrics reporting about file activity
type FilesMetrics struct {
	FileCount *stats.Int64Measure `metric:"fileCount" description:"number of files" extraviews:"sum" tags:"kind,operation"`
	FileSize  *stats.Int64Measure `metric:"fileSize" unit:"bytes" description:"size of files" extraviews:"sum" tags:"kind,operation"`
}

func (f *FilesMetrics) tags(operation string) map[string]string {
	return map[string]string{"kind": "dataset", "operation": operation}
}

// Inc increments the counter for files
func (f *FilesMetrics) Inc(operation string) {
	Inc(f.FileCount, f.tags(operation))
}

// Size measures the size of a file
func (f *FilesMetrics) Size(size int64, operation string) {
	Int64(f.FileSize, size, f.tags(operation))
}

// IOMetrics is a common set of metrics reporting about IO activity
type IOMetrics struct {
	Count        *stats.Int64Measure   `metric:"ioCount" description:"number of IO requests" tags:"kind,operation"`
	Timing       *stats.Float64Measure `metric:"timing" unit:"milliseconds" description:"response time in milliseconds" tags:"kind,operation"`
	Failures     *stats.Int64Measure   `metric:"ioFailures" description:"number of failed IOs" tags:"kind,operation"`
	IOSize       *stats.Int64Measure   `metric:"ioSize" unit:"bytes" description:"IO chunk size in bytes" extraviews:"sum" tags:"kind,operation"`
	IOThroughput *stats.Float64Measure `metric:"throughput" unit:"bytespersec" description:"distribution of throughput of an unitary operation in bytes per second" tags:"kind,operation"`
}

func (n *IOMetrics) tags(operation string) map[string]string {
	return map[string]string{"kind": "io", "operation": operation}
}

// IO records some metrics for an IO operation.
//
// Example:
//
//	var myIOMetrics = &IOMetrics{}
//
//	func (m *myType) MyInstrumentedFunc() {
//	  var size int, err error
//
//	  defer myIOMetrics.IO(time.Now(), "read")
//	  ...
//	  size,err := doSomeWork()
//	  if err != nil {
//	    myIOMetrics.Failed("read")
//	    return err
//	  }
//	  myIOMetrics.IOSize(size, "read")
//	}
func (n *IOMetrics) IO(start time.Time, operation string) {
	now := time.Now()
	Duration(start, now, n.Timing, n.tags(operation))
	Inc(n.Count)
}

// Size records the size of some IO operation. Zero sizes are not recorded.
func (n *IOMetrics) Size(size int64, operation string) {
	if size == 0 {
		return
	}
	Int64(n.IOSize, size, n.tags(operation))
}

// Failed records a failure on some IO operation
func (n *IOMetrics) Failed(operation string) {
	Inc(n.Failures, n.tags(operation))
}

// Throughput records a throuput on a successful, non-empty, IO operation. Expressed in bytes per second.
func (n *IOMetrics) Throughput(start, end time.Time, size int64, operation string) {
	if size == 0 {
		return
	}
	elapsed := end.Sub(start)
	if elapsed == 0 {
		return
	}
	rate := float64(size) / (float64(elapsed) / 1e9)
	Float64(n.IOThroughput, rate, n.tags(operation))
}

// IORecord records all metrics for an IO operation in one go.
//
// It provides an alternative way to record size and error in a single
// deferred call.
//
// Example with deferred error capture:
//
//	var myIOMetrics = &IOMetrics{}
//
//	func (m *myType) MyInstrumentedFunc() {
//	  var size int, err error
//
//	  defer func(start time.Time) {
//	    myIOMetrics.IORecord(start, "read")(size, err)
//	  }(time.Now())
//	  ...
//	  size, err = doSomeWork()
//	  if err != nil {
//	    return
//	  }
//	}
func (n *IOMetrics) IORecord(start time.Time, operation string) func(int64, error) {
	return func(size int64, err error) {
		now := time.Now()
		Duration(start, now, n.Timing, n.tags(operation))
		Inc(n.Count, n.tags(operation))
		n.Size(size, operation)
		if err != nil {
			Inc(n.Failures, n.tags(operation))
			return
		}
		n.Throughput(start, now, size, operation)
	}
}

// UsageMetrics is a common set of metrics reporting about usage
type UsageMetrics struct {
	Count    *stats.Int64Measure   `metric:"usageCount" description:"number of calls" tags:"kind,method"`
	Failures *stats.Int64Measure   `metric:"usageFailures" description:"number of failed calls" tags:"kind,method"`
	Timing   *stats.Float64Measure `metric:"timing" unit:"milliseconds" description:"duration of a call" tags:"kind,method"`
}

func (u *UsageMetrics) tags(method string) map[string]string {
	return map[string]string{"kind": "usage", "method": method}
}

// Inc records the usage of some method, without timings or failure reporting
func (u *UsageMetrics) Inc(method string) {
	Inc(u.Count, u.tags(method))
}

// Used records usage of some instrumented entry point.
//
// Example:
//
//	var myUsageMetrics = &UsageMetrics{}
//
//	func (m *myType) MyInstrumentedFunc() {
//	  defer myUsageMetrics.Used(time.Now(), "MyInstrumentedFunc")
//	  ...
//	  err := doSomeWork()
//	  if err != nil {
//	    myUsageMetrics.Failed()
//	    ...
//	  }
//	}
func (u *UsageMetrics) Used(start time.Time, method string) {
	Since(start, u.Timing, u.tags(method))
	Inc(u.Count, u.tags(method))
}

// UsedAll records usage of some instrumented entry point with failures, in one go.
//
// Example:
//
//	var myUsageMetrics = &UsageMetrics{}
//	var err error
//
//	func (m *myType) MyInstrumentedFunc() {
//	  defer func(start time.Time) {
//	    myUsageMetrics.UsedAll(start, "MyInstrumentedFunc")(err)
//	  }(time.Now())
//	  ...
//	  err = doSomeWork()
//	  if err != nil {
//	    return
//	  }
//	}
func (u *UsageMetrics) UsedAll(start time.Time, method string) func(error) {
	return func(err error) {
		Since(start, u.Timing, u.tags(method))
		Inc(u.Count, u.tags(method))
		if err != nil {
			Inc(u.Failures, u.tags(method))
			return
		}
	}
}

// Failed records a failure on some instrumented entry point
func (u *UsageMetrics) Failed(method string) {
	Inc(u.Failures, u.tags(method))
}

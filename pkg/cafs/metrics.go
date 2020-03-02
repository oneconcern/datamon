package cafs

import (
	"strconv"

	"github.com/oneconcern/datamon/pkg/metrics"
	"go.opencensus.io/stats"
)

// M describes metrics for the cafs package
type M struct {
	Volume struct {
		Blobs cafsMetrics       `group:"blobs" description:"metrics about cafs chunks (blobs)"`
		IO    metrics.IOMetrics `group:"io" description:"metrics about cafs IO operations"`
		Cache cacheUsage        `group:"cache" description:"metrics about cafs ReadAt cache (fuse mount)"`
	} `group:"volumetry" description:""`
	Usage metrics.UsageMetrics `group:"telemetry" description:"usage stats for the cafs package"`

	// more metrics here
}

type cafsMetrics struct {
	BlobsCount     *stats.Int64Measure `metric:"blobs" extraviews:"sum" tags:"kind,operation" description:"number of cafs chunks (blobs), including duplicates"`
	DuplicateCount *stats.Int64Measure `metric:"duplicateBlobs" extraviews:"sum" tags:"kind,operation" description:"number of deduplicated cafs chunks (blobs)"`
	RootsCount     *stats.Int64Measure `metric:"roots" extraviews:"sum" tags:"kind,operation" description:"number of root keys"`
	BlobSize       *stats.Int64Measure `metric:"blobsSize" unit:"sumbytes" tags:"kind,operation" description:"cumulated size of cafs chunks (blobs)"`
}

func (*cafsMetrics) tags(operation string) map[string]string {
	return map[string]string{"kind": "io", "operation": operation}
}

func (m *cafsMetrics) IncBlob(operation string) {
	metrics.Inc(m.BlobsCount, m.tags(operation))
}

func (m *cafsMetrics) IncDuplicate(operation string) {
	metrics.Inc(m.DuplicateCount, m.tags(operation))
}

func (m *cafsMetrics) Size(size int64, operation string) {
	metrics.Int64(m.BlobSize, size, m.tags(operation))
}

func (m *cafsMetrics) IntRoot(keys int, operation string) {
	metrics.Int64(m.RootsCount, int64(keys), m.tags(operation))
}

type cacheUsage struct {
	CacheBuffers     *stats.Int64Measure `metric:"buffers" description:"cache size in buffers the size of a leaf" tags:"leafsize"`
	FreeListHWM      *stats.Int64Measure `metric:"freelistHWM" description:"free list highwater mark" tags:"leafsize"`
	AllocatedBuffers *stats.Int64Measure `metric:"allocated" description:"number of allocated buffers" tags:"leafsize"`
	FreeBuffers      *stats.Int64Measure `metric:"free" description:"number of free buffers" tags:"leafsize"`
	TotalRequested   *stats.Int64Measure `metric:"totalRequested" unit:"bytes" description:"size of the I/O request" tags:"leafsize,operation"`
	TotalRead        *stats.Int64Measure `metric:"totalRead" unit:"bytes" description:"size of the I/O response" tags:"leafsize,operation"`
	RequestedLeaves  *stats.Int64Measure `metric:"leavesRequested" description:"number of leaves requested to complete the operation" tags:"leafsize,operation"`
	FetchedLeaves    *stats.Int64Measure `metric:"leavesFetched" description:"number of leaves actually fetched to complete the operation" tags:"leafsize,operation"`
	WastedLeaves     *stats.Int64Measure `metric:"leavesWasted" description:"number of leaves fetched and wasted because of fast cache recycling" tags:"leafsize,operation"`
	CacheHits        *stats.Int64Measure `metric:"cacheHits" tags:"leafsize,operation"`
	CacheMisses      *stats.Int64Measure `metric:"cacheMisses" tags:"leafsize,operation"`
}

func (u *cacheUsage) tags(leafsize uint32, operation string) map[string]string {
	if operation == "" {
		return map[string]string{"leafsize": strconv.FormatUint(uint64(leafsize), 10)}
	}
	return map[string]string{"leafsize": strconv.FormatUint(uint64(leafsize), 10), "operation": operation}
}

func (u *cacheUsage) Sizing(buffers, freelistHWM int, leafsize uint32) {
	tags := u.tags(leafsize, "")
	metrics.Int64(u.CacheBuffers, int64(buffers), tags)
	metrics.Int64(u.FreeListHWM, int64(freelistHWM), tags)
}

func (u *cacheUsage) Capture(bytesToRead, readBytes int, requested, fetched, wasted, cacheHits, cacheMisses uint64, leafsize uint32, operation string) {
	tags := u.tags(leafsize, operation)
	metrics.Int64(u.TotalRequested, int64(bytesToRead), tags)
	metrics.Int64(u.TotalRead, int64(readBytes), tags)
	metrics.Int64(u.RequestedLeaves, int64(readBytes), tags)
	metrics.Int64(u.FetchedLeaves, int64(fetched), tags)
	metrics.Int64(u.WastedLeaves, int64(wasted), tags)
	metrics.Int64(u.CacheHits, int64(cacheHits), tags)
	metrics.Int64(u.CacheMisses, int64(cacheMisses), tags)
}

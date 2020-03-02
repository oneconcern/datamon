package metrics

import "go.opencensus.io/stats"

type exampleMetrics struct {
	Telemetry struct {
		UsageCounts   []FilesMetrics        `group:"usage" description:""`    // ignored
		FailureCounts []*stats.Int64Measure `group:"failures" description:""` // ignored
		TestCount     *stats.Int64Measure   `metric:"testCount" description:"number of tests"`
	} `group:"telemetry" description:""`
	Volumetry struct {
		Metadata FilesMetrics `group:"metadata" description:""`
	} `group:"volumetry" description:""`
	Network struct {
		Requests IOMetrics
	} `group:"network" description:""`
}

func (e *exampleMetrics) IncTest() {
	Inc(e.Telemetry.TestCount, map[string]string{"kind": "test"})
}

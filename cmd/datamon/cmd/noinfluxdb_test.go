// +build !influxdbintegration

package cmd

func testMetricsEnabled() *bool {
	b := false
	return &b
}

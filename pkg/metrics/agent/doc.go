// Package agent provides an in-process opencensus agent
// to scrape exported metrics and push them to some remote
// open census collector.
//
// The in-process agent is intended for local use of datamon.
//
// For kubernetes deployments running datamon, it is best to
// deploy a standalone opencensus agent container (as sidecar or daemonset).
package agent

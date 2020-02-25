package mocks

import (
	"go.opencensus.io/stats/view"
	"go.uber.org/zap"
)

// NewExporter builds a new mock opencensus exporter
func NewExporter() *Exporter {
	l, _ := zap.NewDevelopment()
	return &Exporter{
		l: l,
	}
}

var _ view.Exporter = &Exporter{}

// Exporter is a mocked up opencensus exporter
type Exporter struct {
	l *zap.Logger
}

// ExportView logs the view data for test purpose
func (e *Exporter) ExportView(viewData *view.Data) {
	e.l.Debug("MockExporter", zap.Any("data", viewData))
}

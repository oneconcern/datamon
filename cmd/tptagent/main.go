package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/oneconcern/pipelines/pkg/cli/envk"
	"github.com/oneconcern/pipelines/pkg/httpd"
	"github.com/oneconcern/pipelines/pkg/log"
	"github.com/oneconcern/pipelines/pkg/tracing"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/spf13/pflag"
	jprom "github.com/uber/jaeger-lib/metrics/prometheus"
	"go.uber.org/zap"
)

var (
	jAgentHostPort string
	baseDir        string
)

func init() {
	httpd.RegisterFlags(pflag.CommandLine)
	pflag.StringVar(
		&jAgentHostPort,
		"jaeger-agent",
		envk.StringOrDefault("JAEGER_HOST", "jaeger-agent:6831"),
		"String representing jaeger-agent host:port",
	)
	pflag.StringVarP(&baseDir, "dir", "d", "/var/lib/trumpet", "the directory for the blobs to be stored")
}

type zapLogger struct {
	lg log.Logger
}

func (z *zapLogger) Printf(format string, args ...interface{}) {
	z.lg.Info(fmt.Sprintf(format, args...))
}

func (z *zapLogger) Fatalf(format string, args ...interface{}) {
	z.lg.Fatal(fmt.Sprintf(format, args...))
}

func main() {
	pflag.Parse()

	lc := zap.NewProductionConfig()
	lc.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	zlg, err := lc.Build()
	if err != nil {
		//#nosec
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}

	logger := log.NewFactory(zlg.With(zap.String("service", "tptagent")))

	tr, closer, err := tracing.Init("tptagent", jprom.New(), logger, jAgentHostPort)
	if err != nil {
		logger.Bg().Info("failed to initialize tracing, falling back to noop tracer", zap.Error(err))
		tr = &opentracing.NoopTracer{}
	}
	if closer != nil {
		defer closer.Close()
	}

	mux := tracing.NewServeMux(tr)
	mux.Handle("/", http.NotFoundHandler())

	httpd.New(
		httpd.LogsWith(&zapLogger{lg: logger.Bg()}),
		httpd.HandlesRequestsWith(mux),
	)
}

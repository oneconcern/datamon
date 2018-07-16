package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/justinas/alice"
	"github.com/oneconcern/pipelines/pkg/cli/envk"
	"github.com/oneconcern/pipelines/pkg/log"
	"github.com/oneconcern/pipelines/pkg/tracing"
	"github.com/oneconcern/trumpet/pkg/httpd"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	jprom "github.com/uber/jaeger-lib/metrics/prometheus"
	"go.uber.org/zap"
)

var (
	jAgentHostPort string
)

func init() {
	httpd.RegisterFlags(pflag.CommandLine)
	pflag.StringVarP(
		&jAgentHostPort,
		"jaeger-agent", "a",
		envk.StringOrDefault("JAEGER_HOST", "jaeger-agent:6831"),
		"String representing jaeger-agent host:port",
	)
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

	lc := zap.NewDevelopmentConfig()
	lc.DisableStacktrace = true
	zlg, err := lc.Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}

	logger := log.NewFactory(zlg.With(zap.String("service", "trumpetd")))

	tr, err := tracing.Init("trumpetd", jprom.New(), logger, jAgentHostPort)
	if err != nil {
		logger.Bg().Warn("failed to initialize tracing, falling back to noop tracer", zap.Error(err))
		tr = &opentracing.NoopTracer{}
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	handler := alice.New(
		requestTracing(tr),
	).Then(mux)

	server := httpd.New(
		httpd.Logger(&zapLogger{lg: logger.Bg()}),
		httpd.RequestHandler(handler),
	)

	if err := server.Listen(); err != nil {
		logger.Bg().Fatal("", zap.Error(err))
	}

	if err := server.Serve(); err != nil {
		logger.Bg().Fatal("", zap.Error(err))
	}
}

func requestTracing(tracer opentracing.Tracer) alice.Constructor {
	return func(next http.Handler) http.Handler {
		return tracing.NewMiddleware(tracer, next)
	}
}

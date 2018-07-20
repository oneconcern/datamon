package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/justinas/alice"
	"github.com/oneconcern/pipelines/pkg/cli/envk"
	"github.com/oneconcern/pipelines/pkg/log"
	"github.com/oneconcern/pipelines/pkg/tracing"
	"github.com/oneconcern/trumpet/pkg/engine"
	"github.com/oneconcern/trumpet/pkg/graphapi"
	"github.com/oneconcern/trumpet/pkg/httpd"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	jprom "github.com/uber/jaeger-lib/metrics/prometheus"
	gqlhandler "github.com/vektah/gqlgen/handler"
	gqltracing "github.com/vektah/gqlgen/opentracing"
	"go.uber.org/zap"
)

var (
	jAgentHostPort string
	baseDir        string
)

func init() {
	httpd.RegisterFlags(pflag.CommandLine)
	pflag.StringVarP(
		&jAgentHostPort,
		"jaeger-agent", "a",
		envk.StringOrDefault("JAEGER_HOST", "jaeger-agent:6831"),
		"String representing jaeger-agent host:port",
	)
	pflag.StringVar(&baseDir, "base-dir", ".trumpet", "the base directory for the database")
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

	logger := log.NewFactory(zlg.With(zap.String("service", "trumpetd")))

	tr, err := tracing.Init("trumpetd", jprom.New(), logger, jAgentHostPort)
	if err != nil {
		logger.Bg().Warn("failed to initialize tracing, falling back to noop tracer", zap.Error(err))
		tr = &opentracing.NoopTracer{}
	}

	eng, err := engine.New(baseDir)
	if err != nil {
		logger.Bg().Fatal("initializing engine", zap.Error(err))
	}

	mux := http.NewServeMux()
	mux.Handle("/", gqlhandler.Playground("Trumpet Server", "/query"))
	mux.Handle("/query", gqlhandler.GraphQL(
		graphapi.NewExecutableSchema(graphapi.NewResolvers(eng)),
		gqlhandler.RequestMiddleware(gqltracing.RequestMiddleware()),
		gqlhandler.ResolverMiddleware(gqltracing.ResolverMiddleware()),
	))
	mux.Handle("/metrics", promhttp.Handler())

	handler := alice.New(
		requestTracing(tr),
	).Then(mux)

	server := httpd.New(
		httpd.LogsWith(&zapLogger{lg: logger.Bg()}),
		httpd.HandlesRequestsWith(handler),
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

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/oneconcern/datamon"

	"github.com/oneconcern/pipelines/pkg/cli/envk"
	"github.com/oneconcern/pipelines/pkg/httpd"
	"github.com/oneconcern/pipelines/pkg/log"
	"github.com/oneconcern/pipelines/pkg/tracing"
	"github.com/oneconcern/datamon/pkg/engine"
	"github.com/oneconcern/datamon/pkg/graphapi"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	tracelog "github.com/opentracing/opentracing-go/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	jprom "github.com/uber/jaeger-lib/metrics/prometheus"
	"github.com/vektah/gqlgen/graphql"
	gqlhandler "github.com/vektah/gqlgen/handler"

	// gqltracing "github.com/vektah/gqlgen/opentracing"
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
	pflag.StringVar(&baseDir, "base-dir", ".datamon", "the base directory for the database")
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
	zlg, err := lc.Build(zap.AddCallerSkip(1))
	if err != nil {
		//#nosec
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}

	logger := log.NewFactory(zlg.With(zap.String("service", "datamond")))

	tr, closer, err := tracing.Init("datamond", jprom.New(), logger, jAgentHostPort)
	if err != nil {
		logger.Bg().Info("failed to initialize tracing, falling back to noop tracer", zap.Error(err))
		tr = &opentracing.NoopTracer{}
	}
	if closer != nil {
		defer closer.Close()
	}

	cfg := datamon.NewConfig(tr, logger)
	cfg.Metadata = baseDir
	eng, err := engine.New(cfg)
	if err != nil {
		logger.Bg().Fatal("initializing engine", zap.Error(err))
	}

	mux := tracing.NewServeMux(tr)
	// mux := http.NewServeMux()
	mux.Handle("/", gqlhandler.Playground("Trumpet Server", "/query"))
	mux.Handle("/query", gqlhandler.GraphQL(
		graphapi.NewExecutableSchema(graphapi.NewResolvers(eng)),
		gqlhandler.ResolverMiddleware(func(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
			rc := graphql.GetResolverContext(ctx)

			parent := opentracing.SpanFromContext(ctx)
			spanOpts := []opentracing.StartSpanOption{
				opentracing.Tag{Key: "resolver.object", Value: rc.Object},
				opentracing.Tag{Key: "resolver.field", Value: rc.Field.Name},
			}
			if parent != nil {
				spanOpts = append(spanOpts, opentracing.ChildOf(parent.Context()))
			}

			span := tr.StartSpan("GQL "+rc.Object+"  "+rc.Field.Name, spanOpts...)
			defer span.Finish()

			ext.SpanKind.Set(span, "server")
			ext.Component.Set(span, "gqlgen")

			res, err = next(ctx)
			if err != nil {
				ext.Error.Set(span, true)
				span.LogFields(
					tracelog.String("event", "error"),
					tracelog.String("message", err.Error()),
					tracelog.String("error.kind", fmt.Sprintf("%T", err)),
				)
			}
			return res, err
		}),
	))

	handler := http.NewServeMux()
	handler.Handle("/metrics", promhttp.Handler())
	handler.HandleFunc("/healthz", healthzEndpoint)
	handler.HandleFunc("/readyz", readyzEndpoint)
	handler.Handle("/", mux)

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

func healthzEndpoint(rw http.ResponseWriter, r *http.Request) {
	rw.Write([]byte("OK"))
}

func readyzEndpoint(rw http.ResponseWriter, r *http.Request) {
	rw.Write([]byte("OK"))
}

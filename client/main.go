package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/honeycombio/opentelemetry-exporter-go/honeycomb"
	otelhttp "go.opentelemetry.io/contrib/instrumentation/net/http"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/exporters/trace/jaeger"
	"go.opentelemetry.io/otel/label"
	export "go.opentelemetry.io/otel/sdk/export/trace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv"
)

type Server struct {
	Tracer trace.Tracer
}

func New() Server {
	return Server{Tracer: global.Tracer("Client")}
}

func initTracer(exporter export.SpanSyncer) {
	// For the demonstration, use sdktrace.AlwaysSample sampler to sample all traces.
	// In a production application, use sdktrace.ProbabilitySampler with a desired probability.
	tp, err := sdktrace.NewProvider(sdktrace.WithConfig(sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),
		sdktrace.WithSyncer(exporter))
	if err != nil {
		log.Fatal(err)
	}
	global.SetTraceProvider(tp)
}

// initTracer creates a new trace provider instance and registers it as global trace provider.
func initJaegerExporter() *jaeger.Exporter {
	exporter, err := jaeger.NewRawExporter(jaeger.WithCollectorEndpoint("http://localhost:32773/api/traces"),
		jaeger.WithProcess(jaeger.Process{
			ServiceName: "trace-demo-client",
			Tags: []label.KeyValue{
				label.String("exporter", "jaeger"),
				label.Float64("float", 312.23),
			},
		}))
	if err != nil {
		return nil
	}

	return exporter
}

func initHoneycombExporter() *honeycomb.Exporter {
	exporter, err := honeycomb.NewExporter(
		honeycomb.Config{
			APIKey: "",
		},
		honeycomb.TargetingDataset(""),
		honeycomb.WithServiceName("opentelemetry-client"),
	)
	if err != nil {
		return nil
	}

	return exporter
}

func main() {
	exporter := initJaegerExporter()
	defer exporter.Flush()

	initTracer(exporter)

	tr := New()
	tr.bar(context.Background())
}

func (s *Server) bar(ctx context.Context) {
	ctx, span := s.Tracer.Start(ctx, "Function Bar", trace.WithAttributes(semconv.PeerServiceKey.String("Client")))
	defer span.End()

	c := &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}

	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:8080/toto", nil)
	if err != nil {
		return
	}

	resp, err := c.Do(req)
	if err != nil {
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	_ = resp.Body.Close()

	fmt.Println(string(body))
}

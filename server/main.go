// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Command jaeger is an example program that creates spans
// and uploads to Jaeger.
package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/honeycombio/opentelemetry-exporter-go/honeycomb"
	otelhttp "go.opentelemetry.io/contrib/instrumentation/net/http"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"

	"go.opentelemetry.io/otel/exporters/trace/jaeger"
	export "go.opentelemetry.io/otel/sdk/export/trace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type Server struct {
	Tracer trace.Tracer
}

func New() Server {
	return Server{Tracer: global.Tracer("Server")}
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
			ServiceName: "trace-demo-server",
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
		honeycomb.WithServiceName("opentelemetry-server"),
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

	tracer := New()

	http.Handle("/toto", otelhttp.NewHandler(http.HandlerFunc(tracer.toto), "Handler Toto"))
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func (s *Server) toto(_ http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	span := trace.SpanFromContext(ctx)
	defer span.End()

	span.AddEvent(ctx, "Printing toto..")
	fmt.Println("toto")

	time.Sleep(1 * time.Millisecond)

	span.AddEvent(ctx, "pass sleep")
	fmt.Println("done")
}

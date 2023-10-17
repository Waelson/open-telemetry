package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/icrowley/fake"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"log"
	"math/rand"
	"time"

	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func initTracerAuto() func(context.Context) error {

	exporter, err := otlptrace.New(
		context.Background(),
		otlptracegrpc.NewClient(
			otlptracegrpc.WithInsecure(),
			otlptracegrpc.WithEndpoint("otel-collector:4317"),
		),
	)

	if err != nil {
		log.Fatal("Could not set exporter: ", err)
	}
	resources, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			attribute.String("service.name", "email-service"),
			attribute.String("application", "app-email"),
		),
	)
	if err != nil {
		log.Fatal("Could not set resources: ", err)
	}

	otel.SetTracerProvider(
		sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
			sdktrace.WithSpanProcessor(sdktrace.NewBatchSpanProcessor(exporter)),
			sdktrace.WithSyncer(exporter),
			sdktrace.WithResource(resources),
		),
	)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return exporter.Shutdown
}

func main() {
	rand.Seed(time.Now().UnixNano())

	cleanup := initTracerAuto()
	defer cleanup(context.Background())

	r := gin.Default()
	r.Use(otelgin.Middleware("app-email"))

	r.GET("/api/v1/email", func(c *gin.Context) {
		t := rand.Intn(9) + 1
		time.Sleep(time.Duration(t) * time.Millisecond)
		email := fake.EmailAddress()
		c.String(200, email)
	})

	// Run the server
	r.Run(":8082")
}

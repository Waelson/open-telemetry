package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"io/ioutil"
	"log"
	"net/http"

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
			attribute.String("service.name", "hello-service"),
			attribute.String("application", "app-hello"),
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
	return exporter.Shutdown
}

func main() {

	cleanup := initTracerAuto()
	defer cleanup(context.Background())

	tracer := otel.Tracer("demo1TracerName")

	otel.SetTextMapPropagator(propagation.TraceContext{})

	r := gin.Default()
	r.Use(otelgin.Middleware("hello-service"))

	r.GET("/", func(c *gin.Context) {
		ctx := context.Background()
		nome, err := getNome(ctx, tracer)
		if err != nil {
			c.String(500, "Erro ao obter o nome")
			return
		}
		c.String(200, fmt.Sprintf("Ola, %s", nome))
	})

	// Run the server
	r.Run(":8080")
}

func getNome(ctx context.Context, tracer trace.Tracer) (string, error) {
	ctx, parentSpan := tracer.Start(
		ctx,
		"get_nome",
		trace.WithAttributes(attribute.String("parentAttributeKey1", "parentAttributeValue1")))

	parentSpan.AddEvent("get_nome-event")
	log.Printf("In parent span, before calling a child function.")

	defer parentSpan.End()

	//
	req, _ := http.NewRequest("GET", "http://app-user:8081/", nil)
	cont := req.Context()
	propagator := otel.GetTextMapPropagator()
	propagator.Inject(cont, propagation.HeaderCarrier(req.Header))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Erro ao fazer a requisição: %v", err)
		return "", err
	}
	defer resp.Body.Close() // Não esqueça de fechar o corpo da resposta!

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Erro ao ler o corpo da resposta: %v", err)
		return "", err
	}

	bodyString := string(bodyBytes)

	return bodyString, nil
	
}

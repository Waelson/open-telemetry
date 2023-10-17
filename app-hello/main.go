package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"time"

	resource2 "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func initTracerAuto() func(context.Context) error {

	client := otlptracegrpc.NewClient(
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint("otel-collector:4317"))

	exporter, err := otlptrace.New(context.Background(), client)

	if err != nil {
		log.Fatal("Could not set exporter: ", err)
	}
	resources, err := resource2.New(
		context.Background(),
		resource2.WithAttributes(
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

	rand.Seed(time.Now().UnixNano())

	cleanup := initTracerAuto()
	defer cleanup(context.Background())

	otel.SetTextMapPropagator(propagation.TraceContext{})
	tracer := otel.Tracer("app-hello")

	r := gin.Default()
	r.Use(otelgin.Middleware("hello-service"))

	r.GET("/api/v1/greeting", func(c *gin.Context) {
		ctx := baggage.ContextWithoutBaggage(c.Request.Context())
		nome, err := getNome(ctx, tracer)

		if err != nil {
			c.String(500, "Erro ao obter o nome")
			return
		}

		email, err := getEmail(ctx, tracer)
		if err != nil {
			c.String(500, "Erro ao obter o email")
			return
		}

		c.String(200, fmt.Sprintf("Ola, %s (%s)", nome, email))
	})

	// Run the server
	r.Run(":8080")
}

func getNome(ctx context.Context, tracer trace.Tracer) (string, error) {
	ctx, span := tracer.Start(ctx, "getNome")
	defer span.End()

	t := rand.Intn(9) + 1
	time.Sleep(time.Duration(t) * time.Millisecond)

	clientHttp := http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://app-user:8081/api/v1/person", nil)
	resp, err := clientHttp.Do(req)
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

func getEmail(ctx context.Context, tracer trace.Tracer) (string, error) {
	ctx, span := tracer.Start(ctx, "getEmail")
	defer span.End()

	t := rand.Intn(9) + 1
	time.Sleep(time.Duration(t) * time.Millisecond)

	clientHttp := http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://app-email:8082/api/v1/email", nil)
	resp, err := clientHttp.Do(req)
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

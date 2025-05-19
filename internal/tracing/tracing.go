package tracing

import (
	"context"
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/shoksin/financesBot/internal/clients/tg"
	"github.com/shoksin/financesBot/internal/logger"
	"github.com/shoksin/financesBot/internal/models/messages"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer

func init() {
	// Инициализация OpenTelemetry + Jaeger
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint("localhost:4318"),
		otlptracehttp.WithInsecure(),
	)

	if err != nil {
		logger.Fatal("Failed to create OTLP exporter", "err", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("tg"),
		)),
	)

	otel.SetTracerProvider(tp)
	tracer = otel.Tracer("tg-bot")
}

func TracingMiddleware(next tg.HandlerFunc) tg.HandlerFunc {
	handler := tg.HandlerFunc(func(tgUpdate tgbotapi.Update, c *tg.Client, msgModel *messages.Model) {
		ctx, span := tracer.Start(msgModel.GetCtx(), "ProcessingMessages")
		defer span.End()

		span.SetAttributes(
			attribute.String("chat.id", fmt.Sprintf("%d", tgUpdate.Message.Chat.ID)),
			attribute.String("message.id", fmt.Sprintf("%d", tgUpdate.Message.MessageID)),
		)

		traceID := span.SpanContext().TraceID().String()
		logger.Info("start span trace", "traceId", traceID)

		msgModel.SetCtx(ctx)
		next.RunFunc(tgUpdate, c, msgModel)
	})

	return handler
}

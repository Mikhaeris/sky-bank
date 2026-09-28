package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/mikhaeris/sky-bank/notification_service/internal/config"
	"github.com/mikhaeris/sky-bank/notification_service/internal/handler"
	"github.com/mikhaeris/sky-bank/notification_service/internal/mailer"
	"github.com/mikhaeris/sky-bank/notification_service/internal/service"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/mikhaeris/sky-bank/pkg/kafka"
	"github.com/mikhaeris/sky-bank/pkg/kafkaevents/constant"
	"github.com/mikhaeris/sky-bank/pkg/telemetry"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg := config.GetConfig(logger)

	tracer, cleanup, err := telemetry.NewTracer(context.Background(), cfg.OtlpEndpoint, "notification_service")
	if err != nil {
		logger.Error("new tracer", "error", err)
		os.Exit(1)
	}
	defer cleanup()

	otel.SetTextMapPropagator(propagation.TraceContext{})

	mailer, err := mailer.New(cfg.Smtp.Host, cfg.Smtp.Port, cfg.Smtp.Username, cfg.Smtp.Password, cfg.Smtp.Sender)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	otpHadler := handler.NewOTPHandler(
		tracer,
		logger,
		mailer,
	)

	otpConsumer, err := kafka.NewConsumer(
		logger, cfg.Kafka.Brokers, constant.GroupOTP, constant.TopicOTPRequested,
		kafka.JSONHandler(otpHadler.Handle),
	)
	if err != nil {
		logger.Error("create otp consumer", "error", err)
		os.Exit(1)
	}
	defer otpConsumer.Stop()

	notificationService := service.NewNotificationService(logger, mailer)
	notificationHandler := handler.NewNotificationHandler(logger, notificationService)

	notificationConsumer, err := kafka.NewConsumer(
		logger, cfg.Kafka.Brokers, constant.GroupNotification, constant.TopicNotificationRequested,
		kafka.JSONHandler(notificationHandler.Handle),
	)
	if err != nil {
		logger.Error("create otp consumer", "error", err)
		os.Exit(1)
	}
	defer notificationConsumer.Stop()

	consumers := []*kafka.Consumer{otpConsumer, notificationConsumer}

	var wg sync.WaitGroup
	for _, c := range consumers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := c.Start(ctx); err != nil {
				logger.Error(err.Error())
			}
		}()
	}

	<-ctx.Done()
	wg.Wait()
}

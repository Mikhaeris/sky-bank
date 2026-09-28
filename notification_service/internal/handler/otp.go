package handler

import (
	"context"
	"log/slog"

	"github.com/mikhaeris/sky-bank/notification_service/internal/mailer"
	ke "github.com/mikhaeris/sky-bank/pkg/kafkaevents"
	"go.opentelemetry.io/otel/trace"
)

type OTPHandler struct {
	tracer trace.Tracer
	logger *slog.Logger
	mailer *mailer.Mailer
}

func NewOTPHandler(
	tracer trace.Tracer,
	logger *slog.Logger,
	mailer *mailer.Mailer,
) *OTPHandler {
	return &OTPHandler{
		tracer: tracer,
		logger: logger,
		mailer: mailer,
	}
}

func (h *OTPHandler) Handle(ctx context.Context, event ke.Event[ke.OtpPayload]) error {
	_, span := h.tracer.Start(ctx, "otp.process")
	defer span.End()

	err := h.mailer.Send(event.Payload.Destination, "otp_code.html", event.Payload.Data)
	if err != nil {
		h.logger.Error(err.Error())
		return err
	}
	h.logger.Info("send otp code email", "email", event.Payload.Destination)
	return nil
}

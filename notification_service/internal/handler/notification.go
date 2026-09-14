package handler

import (
	"context"
	"log/slog"

	notificationv1 "github.com/mikhaeris/sky-bank/notification_service/api/notification/v1"
	"github.com/mikhaeris/sky-bank/notification_service/internal/mailer"
	"google.golang.org/protobuf/types/known/emptypb"
)

type NotificationHandler struct {
	notificationv1.UnimplementedNotificationServiceServer
	logger *slog.Logger
	mailer *mailer.Mailer
}

func NewNotificationHandler(logger *slog.Logger, mailer *mailer.Mailer) *NotificationHandler {
	return &NotificationHandler{
		logger: logger,
		mailer: mailer,
	}
}

func (h *NotificationHandler) SendOtpCode(ctx context.Context, in *notificationv1.SendOtpCodeRequest) (*emptypb.Empty, error) {
	data := map[string]any{
		"otpCode": in.OtpCode,
	}

	err := h.mailer.Send(in.Email, "otp_code.html", data)
	if err != nil {
		h.logger.Error(err.Error())
		return nil, err
	}
	h.logger.Info("send otp code email", "email", in.Email)

	return &emptypb.Empty{}, nil
}

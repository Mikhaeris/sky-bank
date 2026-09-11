package handler

import (
	"context"
	"log/slog"

	notificationv1 "github.com/mikhaeris/sky-bank/notification_service/api/notification/v1"
	"github.com/mikhaeris/sky-bank/notification_service/internal/mailer"
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

func (h *NotificationHandler) SendWelcomeMessage(ctx context.Context, in *notificationv1.WelcomeMessageRequest) (*notificationv1.WelcomeMessageResponse, error) {
	data := map[string]any{
		"activationToken": in.ActivationToken,
		"userID":          in.IdentityUuid,
	}

	err := h.mailer.Send(in.Email, "user_welcome.tmpl", data)
	if err != nil {
		h.logger.Error(err.Error())
	}

	return &notificationv1.WelcomeMessageResponse{
		Status: "ok",
	}, nil
}

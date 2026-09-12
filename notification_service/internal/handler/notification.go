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

	err := h.mailer.Send(in.Email, "user_welcome.html", data)
	if err != nil {
		h.logger.Error(err.Error())
		return &notificationv1.WelcomeMessageResponse{
			Status: "bad",
		}, err
	}
	h.logger.Info("send welcome email", "email", in.Email)

	return &notificationv1.WelcomeMessageResponse{
		Status: "ok",
	}, nil
}

func (h *NotificationHandler) SendWelcomeActivatedMessage(ctx context.Context, in *notificationv1.WelcomeActivatedMessageRequest) (*notificationv1.WelcomeActivatedMessageResponse, error) {
	data := map[string]any{}

	err := h.mailer.Send(in.Email, "user_welcome_activated.html", data)
	if err != nil {
		h.logger.Error(err.Error())
		return &notificationv1.WelcomeActivatedMessageResponse{
			Status: "bad",
		}, err
	}
	h.logger.Info("send welcome activated email", "email", in.Email)

	return &notificationv1.WelcomeActivatedMessageResponse{
		Status: "ok",
	}, nil
}

func (h *NotificationHandler) SendResetPasswordMessage(ctx context.Context, in *notificationv1.ResetPasswordMessageRequest) (*notificationv1.ResetPasswordMessageResponse, error) {
	data := map[string]any{
		"resetPasswordToken": in.ResetPasswordToken,
		"userID":             in.IdentityUuid,
	}

	err := h.mailer.Send(in.Email, "user_reset_password.html", data)
	if err != nil {
		h.logger.Error(err.Error())
		return &notificationv1.ResetPasswordMessageResponse{
			Status: "bad",
		}, err
	}
	h.logger.Info("send reset password email", "email", in.Email)

	return &notificationv1.ResetPasswordMessageResponse{
		Status: "ok",
	}, nil
}

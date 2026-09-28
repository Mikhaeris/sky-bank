package handler

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mikhaeris/sky-bank/notification_service/internal/service"
	ke "github.com/mikhaeris/sky-bank/pkg/kafkaevents"
)

type NotificationHandler struct {
	logger              *slog.Logger
	notificationService *service.NotificationService
}

func NewNotificationHandler(
	logger *slog.Logger,
	notificationService *service.NotificationService,
) *NotificationHandler {
	return &NotificationHandler{
		logger:              logger,
		notificationService: notificationService,
	}
}

func (h *NotificationHandler) Handle(ctx context.Context, event ke.Event[ke.NotificationPayload]) error {
	switch event.Payload.NotificationType {
	case ke.NotificationTypeEmail:
		return h.notificationService.SendEmail(event)
	case ke.NotificationTypeSMS:
		return fmt.Errorf("SMS notification not yet implemented")
	case ke.NotificationTypePush:
		return fmt.Errorf("Push notification not yet implemented")
	default:
		return fmt.Errorf("unsupported notification type")
	}
}

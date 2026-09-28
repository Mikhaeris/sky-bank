package service

import (
	"fmt"
	"log/slog"

	"github.com/mikhaeris/sky-bank/notification_service/internal/mailer"
	ke "github.com/mikhaeris/sky-bank/pkg/kafkaevents"
)

type NotificationService struct {
	logger *slog.Logger
	mailer *mailer.Mailer
}

func NewNotificationService(
	logger *slog.Logger,
	mailer *mailer.Mailer,
) *NotificationService {
	return &NotificationService{
		logger: logger,
		mailer: mailer,
	}
}

func (s *NotificationService) SendEmail(event ke.Event[ke.NotificationPayload]) error {
	template := fmt.Sprintf("%s.html", event.Payload.TemplateID)

	err := s.mailer.Send(event.Payload.Destination, template, event.Payload.Data)
	if err != nil {
		s.logger.Error("send new log in email", "error", err)
		return err
	}
	s.logger.Info("send new log in email", "email", event.Payload.Destination)
	return nil

}

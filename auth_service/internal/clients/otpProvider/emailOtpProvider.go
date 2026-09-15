package otpprovider

import (
	"context"
	"fmt"
	"log/slog"

	notificationv1 "github.com/mikhaeris/sky-bank/notification_service/api/notification/v1"
)

type EmailOtpProvider struct {
	logger *slog.Logger
	client notificationv1.NotificationServiceClient
}

func NewEmailOtpProvider(logger *slog.Logger, client notificationv1.NotificationServiceClient) *EmailOtpProvider {
	return &EmailOtpProvider{
		logger: logger,
		client: client,
	}
}

func (p *EmailOtpProvider) SendOtpCode(ctx context.Context, email string, otpCode string) error {
	_, err := p.client.SendOtpCode(ctx, &notificationv1.SendOtpCodeRequest{
		Email:   email,
		OtpCode: otpCode,
	})
	if err != nil {
		return fmt.Errorf("send otp code: %w", err)
	}

	return nil

}

package kafkaevents

import (
	"time"
	"uuid"

	"github.com/mikhaeris/sky-bank/pkg/kafkaevents/constant"
)

type NotificationType string

const (
	NotificationTypeEmail NotificationType = "email"
	NotificationTypeSMS   NotificationType = "sms"
	NotificationTypePush  NotificationType = "push"
)

type OtpPayload struct {
	IdentityID       uuid.UUID        `json:"id"`
	Destination      string           `json:"destination"`
	NotificationType NotificationType `json:"notification_type"`
	Data             map[string]any   `json:"data"`
	ExpiredAt        *time.Time       `json:"expires_at,omitempty"`
}

type NotificationPayload struct {
	IdentityID       uuid.UUID               `json:"id"`
	Destination      string                  `json:"destination"`
	NotificationType NotificationType        `json:"notification_type"`
	TemplateID       constant.TemplateIDType `json:"template_id"`
	Data             map[string]any          `json:"data"`
	ExpiredAt        *time.Time              `json:"expires_at,omitempty"`
}

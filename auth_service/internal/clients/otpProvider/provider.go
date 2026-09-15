package otpprovider

import "context"

type OtpProvider interface {
	SendOtpCode(ctx context.Context, recipient, otpCode string) error
}

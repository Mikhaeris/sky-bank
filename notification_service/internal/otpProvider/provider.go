package otpprovider

import "context"

type Provider interface {
	SendOtpCode(ctx context.Context, destination, otpCode string) error
}

type OtpProvider struct {
	SmsProvider   Provider
	EmailProvider Provider
}

func NewProvider(smsProvider, emailProvider Provider) *OtpProvider {
	return &OtpProvider{
		SmsProvider:   smsProvider,
		EmailProvider: emailProvider,
	}
}

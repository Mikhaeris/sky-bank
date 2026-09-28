package service

import "github.com/mikhaeris/sky-bank/auth_service/internal/apperr"

var (
	ErrInvalidRefreshToken = apperr.New(apperr.Unauthenticated, "INVALID_REFRESH_TOKEN", "invalid refresh token")
	ErrOtpCodeInvalid      = apperr.New(apperr.InvalidArgument, "INVALID_OTP_CODE", "invalid otp code")
	ErrOtpRateLimited      = apperr.New(apperr.RateLimited, "OTP_RATE_LIMITED", "otp rate limit exceeded")
	ErrDestinationInvalid  = apperr.New(apperr.InvalidArgument, "INVALID_DESTINATION", "invalid destination")
	ErrPurposeInvalid      = apperr.New(apperr.InvalidArgument, "INVALID_PURPOSE", "invalid purpose")
	ErrSessionNotFound     = apperr.New(apperr.NotFound, "SESSION_NOT_FOUND", "session not found")
)

package service

import (
	"time"

	"github.com/mikhaeris/sky-bank/pkg/apperr"
)

var (
	ErrInvalidRefreshToken = apperr.New(apperr.Unauthenticated, "INVALID_REFRESH_TOKEN", "invalid refresh token")
	ErrOtpCodeInvalid      = apperr.New(apperr.InvalidArgument, "INVALID_OTP_CODE", "invalid otp code")
	ErrOtpRateLimited      = apperr.New(apperr.RateLimited, "OTP_RATE_LIMITED", "otp rate limit exceeded")
	ErrDestinationInvalid  = apperr.New(apperr.InvalidArgument, "INVALID_DESTINATION", "invalid destination")
	ErrPurposeInvalid      = apperr.New(apperr.InvalidArgument, "INVALID_PURPOSE", "invalid purpose")
	ErrSessionNotFound     = apperr.New(apperr.NotFound, "SESSION_NOT_FOUND", "session not found")
)

type IssueRateLimitError struct {
	AvailableAt time.Time
}

func (e *IssueRateLimitError) Error() string      { return ErrOtpRateLimited.Error() }
func (e *IssueRateLimitError) Unwrap() error      { return ErrOtpRateLimited }
func (e *IssueRateLimitError) RetryAt() time.Time { return e.AvailableAt }

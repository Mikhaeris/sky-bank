package handler

import (
	"github.com/mikhaeris/sky-bank/pkg/apperr"
)

var (
	ErrInvalidID         = apperr.New(apperr.InvalidArgument, "INVALID_ID", "invalid id")
	ErrInvalidOTPChannel = apperr.New(apperr.InvalidArgument, "INVALID_OTP_CHANNEL", "invalid otp channel")
	ErrInvalidOTPPurpose = apperr.New(apperr.InvalidArgument, "INVALID_OTP_PURPOSE", "invalid otp purpose")
)

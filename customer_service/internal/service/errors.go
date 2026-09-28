package service

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrUserAlreadyExistEmail   = status.Error(codes.AlreadyExists, "user with given email already exist")
	ErrTokenInvalid            = status.Error(codes.Unauthenticated, "invalid token")
	ErrHaveNotPermission       = status.Error(codes.PermissionDenied, "have not permission")
	ErrOtpCodeInvalid          = status.Error(codes.InvalidArgument, "invalid otp code")
	ErrInvalidRefreshToken     = status.Error(codes.InvalidArgument, "invalid refresh token")
	ErrProfileNotCompleted     = status.Error(codes.FailedPrecondition, "profile not completed")
	ErrCustomerAlreadyComplete = status.Error(codes.AlreadyExists, "customer already complete")
	ErrWrongOtpCode            = status.Error(codes.InvalidArgument, "wrong otp code")
)

func internalErr(err error) error {
	return status.Errorf(codes.Internal, "%v", err.Error())
}

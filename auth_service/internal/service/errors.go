package service

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrUserAlreadyExistEmail = status.Error(codes.AlreadyExists, "user with given email already exist")
	ErrTokenInvalid          = status.Error(codes.Unauthenticated, "invalid token")
	ErrHaveNotPermission     = status.Error(codes.PermissionDenied, "have not permission")
	ErrOtpCodeInvalid        = status.Error(codes.InvalidArgument, "invalid otp code")
	ErrRecordNotFound        = status.Error(codes.NotFound, "user with given id not found")
	ErrInvalidRefreshToken   = status.Error(codes.InvalidArgument, "invalid refresh token")
)

func internalErr(err error) error {
	return status.Errorf(codes.Internal, "%v", err.Error())
}

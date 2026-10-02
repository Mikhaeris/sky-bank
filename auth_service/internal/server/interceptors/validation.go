package interceptors

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"buf.build/go/protovalidate"
	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
	"github.com/mikhaeris/sky-bank/auth_service/internal/handler"
	"github.com/mikhaeris/sky-bank/auth_service/internal/service"
	"github.com/mikhaeris/sky-bank/pkg/apperr"
	authv1 "github.com/mikhaeris/sky-bank/proto/gen/auth/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

var errInvalidRequest = apperr.New(apperr.InvalidArgument, "INVALID_REQUEST", "invalid request")

func ValidationInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, next grpc.UnaryHandler) (any, error) {
		switch r := req.(type) {
		case *authv1.CompleteAuthenticationRequest:
			if _, err := uuid.Parse(r.ChallengeId); err != nil {
				return nil, handler.ErrInvalidID
			}
		case *authv1.VerifyChallengeRequest:
			if _, err := uuid.Parse(r.ChallengeId); err != nil {
				return nil, handler.ErrInvalidID
			}
		case *authv1.RevokeSessionRequest:
			if _, err := uuid.Parse(r.SessionId); err != nil {
				return nil, handler.ErrInvalidID
			}
		}
		switch r := req.(type) {
		case *authv1.StartAuthenticationRequest:
			r.Email = service.NormalizeDestination(r.Email, domain.OtpChannelEmail)
		case *authv1.CreateChallengeRequest:
			if r.Channel == authv1.OtpChannel_OTP_CHANNEL_EMAIL {
				r.Destination = service.NormalizeDestination(r.Destination, domain.OtpChannelEmail)
			} else {
				r.Destination = service.NormalizeDestination(r.Destination, domain.OtpChannelSMS)
			}
		case *authv1.VerifyChallengeRequest:
			r.Destination = service.NormalizeDestination(r.Destination, domain.OtpChannelEmail)
		}

		message, ok := req.(proto.Message)
		if !ok {
			return nil, fmt.Errorf("validate auth request: unexpected request type %T", req)
		}
		if err := validateMessage(message); err != nil {
			return nil, err
		}

		switch r := req.(type) {
		case *authv1.CreateChallengeRequest:
			if r.Channel == authv1.OtpChannel_OTP_CHANNEL_EMAIL {
				if err := validateMessage(&authv1.StartAuthenticationRequest{Email: r.Destination}); err != nil {
					return nil, err
				}
			}
		case *authv1.VerifyChallengeRequest:
			if err := validateMessage(&authv1.StartAuthenticationRequest{Email: r.Destination}); err != nil {
				return nil, err
			}
		}
		return next(ctx, req)
	}
}

func validateMessage(message proto.Message) error {
	if err := protovalidate.Validate(message); err != nil {
		var validationErr *protovalidate.ValidationError
		if errors.As(err, &validationErr) {
			return errInvalidRequest
		}
		return fmt.Errorf("validate auth request: %w", err)
	}
	return nil
}

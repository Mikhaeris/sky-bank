package interceptors

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/mikhaeris/sky-bank/pkg/apperr"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
)

func grpcCode(kind apperr.Kind) (codes.Code, bool) {
	switch kind {
	case apperr.InvalidArgument:
		return codes.InvalidArgument, true
	case apperr.Unauthenticated:
		return codes.Unauthenticated, true
	case apperr.NotFound:
		return codes.NotFound, true
	case apperr.RateLimited:
		return codes.ResourceExhausted, true
	default:
		return codes.Internal, false
	}
}

func ErrorInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, next grpc.UnaryHandler) (any, error) {
		response, err := next(ctx, req)
		if err == nil {
			return response, nil
		}
		if ctx.Err() != nil {
			return nil, status.FromContextError(ctx.Err()).Err()
		}

		var publicErr *apperr.Error
		if errors.As(err, &publicErr) {
			if code, ok := grpcCode(publicErr.Kind()); ok {
				errorInfo := &errdetails.ErrorInfo{Reason: publicErr.Reason(), Domain: "auth_service"}
				var retryable interface{ RetryAt() time.Time }
				hasRetry := code == codes.ResourceExhausted && errors.As(err, &retryable)
				if hasRetry {
					errorInfo.Metadata = map[string]string{"retry_at": retryable.RetryAt().UTC().Format(time.RFC3339Nano)}
				}
				publicStatus, detailErr := status.New(code, publicErr.Message()).WithDetails(
					errorInfo,
				)
				if detailErr == nil && hasRetry {
					publicStatus, detailErr = publicStatus.WithDetails(&errdetails.RetryInfo{
						RetryDelay: durationpb.New(max(0, time.Until(retryable.RetryAt()))),
					})
				}
				if detailErr == nil {
					return nil, publicStatus.Err()
				}
			}
		}

		if errors.Is(err, context.Canceled) {
			logger.ErrorContext(ctx, "grpc request failed", "method", info.FullMethod, "error", err)
			return nil, status.FromContextError(context.Canceled).Err()
		}
		if errors.Is(err, context.DeadlineExceeded) {
			logger.ErrorContext(ctx, "grpc request failed", "method", info.FullMethod, "error", err)
			return nil, status.FromContextError(context.DeadlineExceeded).Err()
		}
		logger.ErrorContext(ctx, "grpc request failed", "method", info.FullMethod, "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
}

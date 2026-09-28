package server

import (
	"context"
	"errors"
	"log/slog"
	"runtime/debug"

	"github.com/mikhaeris/sky-bank/auth_service/internal/apperr"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func RecoveryInterceptors(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.ErrorContext(ctx, "panic in grpc handler",
					"method", info.FullMethod,
					"panic", recovered,
					"stack", string(debug.Stack()),
				)

				resp = nil
				err = status.Error(
					codes.Internal,
					"internal error",
				)
			}
		}()

		return handler(ctx, req)
	}
}

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
				publicStatus, detailErr := status.New(code, publicErr.Message()).WithDetails(
					&errdetails.ErrorInfo{Reason: publicErr.Reason(), Domain: "auth_service"},
				)
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

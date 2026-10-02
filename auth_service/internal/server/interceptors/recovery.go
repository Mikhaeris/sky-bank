package interceptors

import (
	"context"
	"log/slog"
	"runtime/debug"

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

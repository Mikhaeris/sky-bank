package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"time"

	"google.golang.org/grpc"
)

const gracefulShutdownTimeout = 10 * time.Second

var ErrGRPCShutdownTimeout = errors.New("grpc graceful shutdown timed out")

func RunGRPCServer(ctx context.Context, grpcServer *grpc.Server, addr string, logger *slog.Logger) error {
	if ctx.Err() != nil {
		return nil
	}
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen for grpc: %w", err)
	}

	logger.Info(
		"starting server",
		"addr", addr,
	)

	serveResult := make(chan error, 1)
	go func() {
		serveResult <- grpcServer.Serve(lis)
	}()

	select {
	case err := <-serveResult:
		if err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			_ = stopGRPCServer(grpcServer, logger)
			return fmt.Errorf("serve grpc: %w", err)
		}
	case <-ctx.Done():
		logger.Info("shutting down server", "reason", ctx.Err())
		if err := stopGRPCServer(grpcServer, logger); err != nil {
			return err
		}
		if err := <-serveResult; err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			return fmt.Errorf("serve grpc: %w", err)
		}
	}

	logger.Info("stopped server", "addr", addr)

	return nil
}

func stopGRPCServer(grpcServer *grpc.Server, logger *slog.Logger) error {
	done := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(done)
	}()

	timer := time.NewTimer(gracefulShutdownTimeout)
	defer timer.Stop()

	select {
	case <-done:
		logger.Info("server stopped gracefully")
		return nil
	case <-timer.C:
		logger.Warn("graceful shutdown timeout, forcing stop")
		go grpcServer.Stop()
		return ErrGRPCShutdownTimeout
	}
}

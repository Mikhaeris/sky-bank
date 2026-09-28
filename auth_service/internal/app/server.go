package app

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
)

const (
	shutdownSignalBufferSize = 1
	gracefulShutdownTimeout  = 10 * time.Second
)

func RunGRPCServer(grpcServer *grpc.Server, addr string, logger *slog.Logger) error {
	go func() {
		quit := make(chan os.Signal, shutdownSignalBufferSize)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(quit)

		s := <-quit

		logger.Info(
			"shutting down server",
			"signal", s.String(),
		)

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

		case <-timer.C:
			logger.Warn("graceful shutdown timeout, forcing stop")
			grpcServer.Stop()
		}
	}()

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen for grpc: %w", err)
	}

	logger.Info(
		"starting server",
		"addr", addr,
	)

	if err := grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("serve grpc: %w", err)
	}

	logger.Info("stopped server", "addr", addr)

	return nil
}

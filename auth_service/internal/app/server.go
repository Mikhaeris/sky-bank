package app

import (
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
)

func RunGRPCServer(grpcServer *grpc.Server, addr string, logger *slog.Logger) error {
	go func() {
		quit := make(chan os.Signal, 1)
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

		timer := time.NewTimer(10 * time.Second)
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
		return err
	}

	logger.Info(
		"starting server",
		"addr", addr,
	)

	if err := grpcServer.Serve(lis); err != nil {
		return err
	}

	logger.Info("stopped server", "addr", addr)

	return nil
}

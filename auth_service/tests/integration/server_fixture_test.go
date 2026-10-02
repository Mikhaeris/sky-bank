package integration_test

import (
	"bytes"
	"context"
	"log/slog"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/mikhaeris/sky-bank/auth_service/internal/app"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type blockingHealthServer struct {
	healthpb.UnimplementedHealthServer
	started            chan struct{}
	release            chan struct{}
	ignoreCancellation bool
}

func (s *blockingHealthServer) Check(ctx context.Context, _ *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	close(s.started)
	if s.ignoreCancellation {
		<-s.release
		return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil
	}
	select {
	case <-s.release:
		return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

type lockedLogBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (b *lockedLogBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.Write(p)
}

func (b *lockedLogBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}

type grpcShutdownFixture struct {
	cancel     context.CancelFunc
	release    func()
	result     <-chan error
	callResult <-chan error
	callCtx    context.Context
	logs       *lockedLogBuffer
}

type grpcShutdownOptions struct {
	ignoreRPCCancellation bool
	rpcTimeout            time.Duration
}

func newGRPCShutdownFixture(t *testing.T, options grpcShutdownOptions) *grpcShutdownFixture {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}

	server := grpc.NewServer()
	health := &blockingHealthServer{
		started: make(chan struct{}), release: make(chan struct{}), ignoreCancellation: options.ignoreRPCCancellation,
	}
	healthpb.RegisterHealthServer(server, health)
	release := sync.OnceFunc(func() { close(health.release) })
	ctx, cancel := context.WithCancel(context.Background())
	logs := &lockedLogBuffer{}
	result := make(chan error, 1)
	go func() {
		result <- app.RunGRPCServer(ctx, server, addr, slog.New(slog.NewTextHandler(logs, nil)))
	}()

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		release()
		cancel()
		server.Stop()
		t.Fatal(err)
	}
	callCtx, stopCall := context.WithTimeout(context.Background(), options.rpcTimeout)
	callResult := make(chan error, 1)
	go func() {
		_, err := healthpb.NewHealthClient(conn).Check(callCtx, &healthpb.HealthCheckRequest{}, grpc.WaitForReady(true))
		callResult <- err
	}()
	t.Cleanup(func() {
		release()
		cancel()
		stopCall()
		_ = conn.Close()
		server.Stop()
	})

	select {
	case <-health.started:
	case err := <-result:
		t.Fatalf("server stopped before RPC started: %v; logs: %s", err, logs.String())
	case <-callCtx.Done():
		t.Fatalf("RPC did not start: %v; logs: %s", callCtx.Err(), logs.String())
	}
	return &grpcShutdownFixture{
		cancel: cancel, release: release, result: result,
		callResult: callResult, callCtx: callCtx, logs: logs,
	}
}

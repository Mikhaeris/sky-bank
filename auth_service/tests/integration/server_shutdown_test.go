package integration_test

import (
	"errors"
	"testing"
	"time"

	"github.com/mikhaeris/sky-bank/auth_service/internal/app"
)

func TestRunGRPCServerWaitsForActiveRPC(t *testing.T) {
	f := newGRPCShutdownFixture(t, grpcShutdownOptions{rpcTimeout: 5 * time.Second})
	f.cancel()

	select {
	case err := <-f.result:
		t.Fatalf("server returned while RPC was active: %v; logs: %s", err, f.logs.String())
	case <-time.After(100 * time.Millisecond):
	}

	f.release()
	select {
	case err := <-f.callResult:
		if err != nil {
			t.Fatalf("active RPC failed during graceful shutdown: %v", err)
		}
	case <-f.callCtx.Done():
		t.Fatal("active RPC did not finish")
	}
	select {
	case err := <-f.result:
		if err != nil {
			t.Fatalf("server shutdown failed: %v; logs: %s", err, f.logs.String())
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("server did not return after active RPC finished; logs: %s", f.logs.String())
	}
}

func TestRunGRPCServerForcesStopAfterTimeout(t *testing.T) {
	f := newGRPCShutdownFixture(t, grpcShutdownOptions{
		ignoreRPCCancellation: true,
		rpcTimeout:            15 * time.Second,
	})
	f.cancel()

	select {
	case err := <-f.result:
		if !errors.Is(err, app.ErrGRPCShutdownTimeout) {
			t.Fatalf("expected graceful shutdown timeout, got %v; logs: %s", err, f.logs.String())
		}
	case <-time.After(12 * time.Second):
		t.Fatalf("server did not return after forced shutdown timeout; logs: %s", f.logs.String())
	}
}

package app

import (
	"context"
	"log/slog"
	"time"

	"github.com/mikhaeris/sky-bank/auth_service/internal/service"
)

const (
	otpCleanupTimeout  = 30 * time.Second
	otpCleanupInterval = time.Hour
)

func RunOTPCleanup(ctx context.Context, logger *slog.Logger, challenges *service.ChallengeService) {
	cleanup := func() {
		runCtx, cancel := context.WithTimeout(ctx, otpCleanupTimeout)
		defer cancel()

		expired, inactive, err := challenges.Cleanup(runCtx)
		if err != nil {
			if ctx.Err() == nil {
				logger.Error("OTP cleanup failed", "error", err)
			}
			return
		}
		if expired > 0 || inactive > 0 {
			logger.Info("OTP cleanup completed", "expired_challenges", expired, "inactive_limits", inactive)
		}
	}

	cleanup()
	ticker := time.NewTicker(otpCleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cleanup()
		}
	}
}

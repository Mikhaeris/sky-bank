package unit_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/mikhaeris/sky-bank/auth_service/internal/config"
	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
	"github.com/mikhaeris/sky-bank/auth_service/internal/service"
)

func TestNextIssueState(t *testing.T) {
	now := time.Date(2026, time.September, 29, 12, 0, 0, 0, time.UTC)
	limits := config.OtpLimits{InitialIssues: 5, InitialCooldown: 15 * time.Second, MaxCooldown: 5 * time.Minute, IssueResetAfter: 30 * time.Minute}
	recent := func(count int, elapsed time.Duration) domain.OTPLimitState {
		last := now.Add(-elapsed)
		return domain.OTPLimitState{LastIssuedAt: &last, IssueCount: count}
	}
	waits := []time.Duration{15, 15, 15, 15, 30, 60, 120, 240, 300, 300}
	for i, seconds := range waits {
		count, wait := i+1, seconds*time.Second
		t.Run(fmt.Sprintf("issue %d", count+1), func(t *testing.T) {
			limit := recent(count, wait-time.Nanosecond)
			_, err := service.NextIssueState(now, limit, limits)
			var retry *service.IssueRateLimitError
			if !errors.Is(err, service.ErrOtpRateLimited) || !errors.As(err, &retry) || !retry.RetryAt().Equal(now.Add(time.Nanosecond)) {
				t.Fatalf("unexpected early request error: %v", err)
			}
			next, err := service.NextIssueState(now.Add(time.Nanosecond), limit, limits)
			if err != nil || next.IssueCount != count+1 {
				t.Fatalf("request at deadline: state=%+v err=%v", next, err)
			}
		})
	}
	for _, tt := range []struct {
		name  string
		state domain.OTPLimitState
		count int
	}{
		{"first request", domain.OTPLimitState{}, 1},
		{"later requests remain available", recent(100, 5*time.Minute), 101},
		{"before idle reset", recent(9, 30*time.Minute-time.Nanosecond), 10},
		{"at idle reset", recent(9, 30*time.Minute), 1},
		{"after idle reset", recent(9, time.Hour), 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			next, err := service.NextIssueState(now, tt.state, limits)
			if err != nil || next.IssueCount != tt.count || next.LastIssuedAt == nil || !next.LastIssuedAt.Equal(now) || !next.UpdatedAt.Equal(now) {
				t.Fatalf("state=%+v err=%v", next, err)
			}
		})
	}
	t.Run("rejections keep deadline", func(t *testing.T) {
		limit := recent(9, time.Minute)
		for _, elapsed := range []time.Duration{0, time.Minute, 3 * time.Minute} {
			_, err := service.NextIssueState(now.Add(elapsed), limit, limits)
			var retry *service.IssueRateLimitError
			if !errors.As(err, &retry) || !retry.RetryAt().Equal(now.Add(4*time.Minute)) {
				t.Fatalf("deadline changed: %v", err)
			}
		}
		if limit.IssueCount != 9 || !limit.LastIssuedAt.Equal(now.Add(-time.Minute)) {
			t.Fatal("rejection changed state")
		}
	})
}

func TestNormalizeDestination(t *testing.T) {
	if got := service.NormalizeDestination("  Alice@Example.COM  ", domain.OtpChannelEmail); got != "alice@example.com" {
		t.Fatalf("email key: %q", got)
	}
	if got := service.NormalizeDestination("  +123456789  ", domain.OtpChannelSMS); got != "+123456789" {
		t.Fatalf("sms key: %q", got)
	}
}

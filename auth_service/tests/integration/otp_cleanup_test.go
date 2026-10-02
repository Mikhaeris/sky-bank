package integration_test

import (
	"testing"
	"time"

	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
)

func TestOTPCleanupIntegration(t *testing.T) {
	env := newOTPTestEnv(t)

	t.Run("deletes expired challenges", func(t *testing.T) {
		expired := env.newChallenge("expired@example.invalid")
		expired.ExpiresAt = time.Now().Add(-time.Minute)
		env.saveChallenge(t, expired)

		active := env.newChallenge("active@example.invalid")
		env.saveChallenge(t, active)

		removed, _, err := env.service.Cleanup(env.ctx)
		if err != nil {
			t.Fatal(err)
		}
		if removed != 1 {
			t.Fatalf("removed challenges = %d, want 1", removed)
		}

		var remaining int
		if err := env.pool.QueryRow(env.ctx, "SELECT COUNT(*) FROM challenges WHERE id IN ($1, $2)", expired.ID, active.ID).Scan(&remaining); err != nil {
			t.Fatal(err)
		}
		if remaining != 1 {
			t.Fatalf("remaining challenges = %d, want 1", remaining)
		}
		if _, err := env.store.ChallengeRepository().GetByChallengeId(env.ctx, active.ID); err != nil {
			t.Fatalf("active challenge: %v", err)
		}
	})

	t.Run("deletes inactive issuance limits and keeps recent ones", func(t *testing.T) {
		now := time.Now().UTC()
		for _, limit := range []struct {
			destination string
			updatedAt   time.Time
		}{
			{"stale@example.invalid", now.Add(-25 * time.Hour)},
			{"recent@example.invalid", now.Add(-23 * time.Hour)},
		} {
			if _, err := env.pool.Exec(env.ctx, `INSERT INTO otp_limits (destination, channel, purpose, updated_at)
                VALUES ($1, $2, $3, $4)`, limit.destination, domain.OtpChannelEmail, domain.CodePurposeAuthentication, limit.updatedAt); err != nil {
				t.Fatal(err)
			}
		}
		_, removed, err := env.service.Cleanup(env.ctx)
		if err != nil || removed != 1 {
			t.Fatalf("cleanup: removed=%d err=%v", removed, err)
		}
		var stale, recent bool
		if err := env.pool.QueryRow(env.ctx, `SELECT
            EXISTS (SELECT 1 FROM otp_limits WHERE destination = 'stale@example.invalid'),
            EXISTS (SELECT 1 FROM otp_limits WHERE destination = 'recent@example.invalid')`).Scan(&stale, &recent); err != nil {
			t.Fatal(err)
		}
		if stale || !recent {
			t.Fatalf("unexpected remaining limits: stale=%v recent=%v", stale, recent)
		}
	})
}

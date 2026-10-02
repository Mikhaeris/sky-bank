package integration_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
	"uuid"

	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
	"github.com/mikhaeris/sky-bank/auth_service/internal/repository"
	"github.com/mikhaeris/sky-bank/auth_service/internal/service"
)

func TestOTPLimitsIntegration(t *testing.T) {
	env := newOTPTestEnv(t)

	t.Run("issue and challenge commit or roll back together", func(t *testing.T) {
		challenge := env.newChallenge("issue@example.invalid")
		issuedAt := time.Now().UTC()
		err := env.store.Atomic(env.ctx, func(ctx context.Context, tx repository.TxStore) error {
			limitsRepo := tx.OTPLimitRepository()
			limit, err := limitsRepo.Lock(env.ctx, challenge.Destination, challenge.Channel, challenge.Purpose)
			if err != nil {
				return err
			}
			if limit.IssueCount != 0 || limit.LastIssuedAt != nil ||
				limit.Destination != challenge.Destination || limit.Channel != challenge.Channel ||
				limit.Purpose != challenge.Purpose || limit.UpdatedAt.IsZero() {
				return fmt.Errorf("unexpected initial limit: %+v", limit)
			}
			limit.LastIssuedAt = &issuedAt
			limit.IssueCount = 1
			limit.UpdatedAt = issuedAt
			if err := limitsRepo.RecordIssue(env.ctx, limit); err != nil {
				return err
			}
			_, err = tx.ChallengeRepository().Upster(ctx, challenge)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}

		var count int
		query := "SELECT issue_count FROM otp_limits WHERE destination = $1 AND channel = $2 AND purpose = $3"
		if err := env.pool.QueryRow(env.ctx, query, challenge.Destination, challenge.Channel, challenge.Purpose).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("issue count = %d, want 1", count)
		}
		if _, err := env.store.ChallengeRepository().GetByChallengeId(env.ctx, challenge.ID); err != nil {
			t.Fatal(err)
		}

		rollback := errors.New("rollback test transaction")
		err = env.store.Atomic(env.ctx, func(ctx context.Context, tx repository.TxStore) error {
			limitsRepo := tx.OTPLimitRepository()
			limit, err := limitsRepo.Lock(env.ctx, challenge.Destination, challenge.Channel, challenge.Purpose)
			if err != nil {
				return err
			}
			if limit.IssueCount != 1 {
				return fmt.Errorf("locked issue count = %d, want 1", limit.IssueCount)
			}
			limit.IssueCount = 2
			if err := limitsRepo.RecordIssue(env.ctx, limit); err != nil {
				return err
			}
			return rollback
		})
		if !errors.Is(err, rollback) {
			t.Fatalf("transaction error = %v, want rollback error", err)
		}
		if err := env.pool.QueryRow(env.ctx, query, challenge.Destination, challenge.Channel, challenge.Purpose).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("issue count after rollback = %d, want 1", count)
		}
	})

	t.Run("failures stay within each challenge", func(t *testing.T) {
		destination := "failures@example.invalid"
		for range 3 {
			challenge := env.newChallenge(destination)
			env.saveChallenge(t, challenge)
			assertFiveFailures(t, env, challenge.ID)
			if _, err := env.store.ChallengeRepository().GetByChallengeId(env.ctx, challenge.ID); !errors.Is(err, repository.ErrRecordNotFound) {
				t.Fatalf("challenge after fifth failure: %v", err)
			}
			if err := env.consume(challenge.ID, testCode); !errors.Is(err, service.ErrOtpCodeInvalid) {
				t.Fatalf("exhausted challenge accepted: %v", err)
			}
		}
		var limitsExist bool
		if err := env.pool.QueryRow(env.ctx, "SELECT EXISTS (SELECT 1 FROM otp_limits WHERE destination = $1)", destination).Scan(&limitsExist); err != nil {
			t.Fatal(err)
		}
		if limitsExist {
			t.Fatal("verification created recipient-wide limit state")
		}

		dto := domain.IssueChallengeDTO{Destination: destination, Channel: domain.OtpChannelEmail, Purpose: domain.CodePurposeAuthentication}
		id, err := env.service.Issue(env.ctx, dto)
		if err != nil {
			t.Fatalf("past failures prevented issuance: %v", err)
		}
		if _, err := env.service.Issue(env.ctx, dto); !errors.Is(err, service.ErrOtpRateLimited) {
			t.Fatalf("issuance cooldown ignored: %v", err)
		}
		challenge, err := env.store.ChallengeRepository().GetByChallengeId(env.ctx, id)
		if err != nil || challenge.FailedAttempts != 0 {
			t.Fatalf("new challenge: %+v err=%v", challenge, err)
		}
		var eventID uuid.UUID
		var encrypted []byte
		if err := env.pool.QueryRow(env.ctx, "SELECT event_id, encrypted_event FROM otp_outbox WHERE challenge_id = $1", id).Scan(&eventID, &encrypted); err != nil {
			t.Fatal(err)
		}
		event, err := env.cipher.Decrypt(encrypted, eventID, id)
		if err != nil {
			t.Fatal(err)
		}
		code, ok := event.Payload.Data["otpCode"].(string)
		if !ok {
			t.Fatal("OTP code is missing from the event")
		}
		if err := env.consume(id, code); err != nil {
			t.Fatalf("past failures prevented valid login: %v", err)
		}
	})

	t.Run("concurrent wrong attempts are counted once each", func(t *testing.T) {
		challenge := env.newChallenge("concurrent-wrong@example.invalid")
		env.saveChallenge(t, challenge)
		start := make(chan struct{})
		results := make(chan error, 4)
		for range 4 {
			go func() { <-start; results <- env.consume(challenge.ID, "000000") }()
		}
		close(start)
		for range 4 {
			if err := <-results; !errors.Is(err, service.ErrOtpCodeInvalid) {
				t.Fatalf("wrong attempt: %v", err)
			}
		}
		state, err := env.store.ChallengeRepository().GetByChallengeId(env.ctx, challenge.ID)
		if err != nil || state.FailedAttempts != 4 {
			t.Fatalf("lost attempts: state=%+v err=%v", state, err)
		}
		if err := env.consume(challenge.ID, "000000"); !errors.Is(err, service.ErrOtpRateLimited) {
			t.Fatalf("fifth error: %v", err)
		}
		if err := env.consume(challenge.ID, testCode); !errors.Is(err, service.ErrOtpCodeInvalid) {
			t.Fatalf("exhausted code: %v", err)
		}
	})

	t.Run("valid code can be used once", func(t *testing.T) {
		challenge := env.newChallenge("single-use@example.invalid")
		env.saveChallenge(t, challenge)
		if err := env.consume(challenge.ID, testCode); err != nil {
			t.Fatalf("valid code: %v", err)
		}
		if err := env.consume(challenge.ID, testCode); err != service.ErrOtpCodeInvalid {
			t.Fatalf("reused code: %v", err)
		}
	})

	t.Run("valid code is accepted after four wrong attempts", func(t *testing.T) {
		challenge := env.newChallenge("four-errors@example.invalid")
		env.saveChallenge(t, challenge)
		for range 4 {
			if err := env.consume(challenge.ID, "000000"); !errors.Is(err, service.ErrOtpCodeInvalid) {
				t.Fatalf("wrong attempt: %v", err)
			}
		}
		if err := env.consume(challenge.ID, testCode); err != nil {
			t.Fatalf("valid fifth attempt: %v", err)
		}
	})

	t.Run("concurrent consumption allows one success", func(t *testing.T) {
		challenge := env.newChallenge("concurrent@example.invalid")
		env.saveChallenge(t, challenge)
		results := make(chan error, 2)
		go func() { results <- env.consume(challenge.ID, testCode) }()
		go func() { results <- env.consume(challenge.ID, testCode) }()

		valid, invalid := 0, 0
		for range 2 {
			switch err := <-results; err {
			case nil:
				valid++
			case service.ErrOtpCodeInvalid:
				invalid++
			default:
				t.Fatalf("concurrent consume: %v", err)
			}
		}
		if valid != 1 || invalid != 1 {
			t.Fatalf("concurrent consume: valid=%d invalid=%d; want 1 each", valid, invalid)
		}
	})
}

func assertFiveFailures(t *testing.T, env *otpTestEnv, id uuid.UUID) {
	t.Helper()
	for attempt := 1; attempt <= 5; attempt++ {
		err := env.consume(id, "000000")
		want := service.ErrOtpCodeInvalid
		if attempt == 5 {
			want = service.ErrOtpRateLimited
		}
		if err != want {
			t.Fatalf("attempt %d: got %v, want %v", attempt, err, want)
		}
	}
}

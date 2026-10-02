package integration_test

import (
	"errors"
	"testing"

	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
	"github.com/mikhaeris/sky-bank/auth_service/internal/lib/jwt"
	"github.com/mikhaeris/sky-bank/auth_service/internal/service"
)

func TestCompleteAuthenticationScenarios(t *testing.T) {
	f := newAuthenticationFixture(t)
	t.Run("rolls back on identity, signing, and session failures", f.testRollback)
	t.Run("normalizes email and rejects malformed OTP", f.testNormalization)
	t.Run("counts wrong OTP attempts", f.testWrongAttempts)
	t.Run("concurrent completion creates one session", f.testConcurrentCompletion)
}

func (f *authenticationFixture) testRollback(t *testing.T) {
	for _, stage := range []string{"identity", "signing", "session"} {
		t.Run(stage, func(t *testing.T) {
			email := stage + "@example.com"
			dto := f.issue(t, email)
			failingAuth := f.auth
			restore := func() {}
			if stage == "signing" {
				failingAuth = f.newService(&jwt.Keys{})
			} else {
				table := "identities"
				if stage == "session" {
					table = "sessions"
				}
				if _, err := f.pool.Exec(f.ctx,
					"ALTER TABLE "+table+" ADD CONSTRAINT reject_auth_write CHECK (false) NOT VALID",
				); err != nil {
					t.Fatal(err)
				}
				restored := false
				restore = func() {
					if restored {
						return
					}
					restored = true
					if _, err := f.pool.Exec(f.ctx,
						"ALTER TABLE "+table+" DROP CONSTRAINT reject_auth_write",
					); err != nil {
						t.Error(err)
					}
				}
				t.Cleanup(restore)
			}

			tokens, err := failingAuth.CompleteAuthentication(f.ctx, dto)
			restore()
			if err == nil || errors.Is(err, service.ErrOtpCodeInvalid) {
				t.Fatalf("expected infrastructure failure, got %v", err)
			}
			if tokens != (domain.Tokens{}) {
				t.Fatal("failed transaction returned tokens")
			}
			f.assertState(t, email, dto, authenticationState{challenges: 1, outbox: 1})

			tokens, err = f.auth.CompleteAuthentication(f.ctx, dto)
			if err != nil {
				t.Fatalf("retry with same OTP: %v", err)
			}
			if tokens.Access == "" || tokens.Refresh == "" {
				t.Fatal("missing tokens")
			}
			if _, err := f.store.SessionRepository().GetByRefreshTokenHash(f.ctx, domain.HashRefreshToken(tokens.Refresh)); err != nil {
				t.Fatalf("returned refresh token has no session: %v", err)
			}
			f.assertState(t, email, dto, authenticationState{identities: 1, sessions: 1})
		})
	}
}

func (f *authenticationFixture) testNormalization(t *testing.T) {
	const canonicalEmail = "alice@example.com"
	dto := f.issue(t, "  Alice@Example.COM  ")
	var challengeDestination, limitDestination string
	if err := f.pool.QueryRow(f.ctx,
		`SELECT destination FROM challenges WHERE id = $1`, dto.ChallengeID,
	).Scan(&challengeDestination); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(f.ctx,
		`SELECT destination FROM otp_limits WHERE destination = $1`, canonicalEmail,
	).Scan(&limitDestination); err != nil {
		t.Fatal(err)
	}
	if challengeDestination != canonicalEmail || limitDestination != canonicalEmail {
		t.Fatalf("challenge=%q limit=%q", challengeDestination, limitDestination)
	}
	var outbox domain.OTPOutbox
	if err := f.pool.QueryRow(f.ctx,
		`SELECT event_id, encrypted_event FROM otp_outbox WHERE challenge_id = $1`, dto.ChallengeID,
	).Scan(&outbox.EventID, &outbox.EncryptedEvent); err != nil {
		t.Fatal(err)
	}
	event, err := f.cipher.Decrypt(outbox.EncryptedEvent, outbox.EventID, dto.ChallengeID)
	if err != nil || event.Payload.Destination != canonicalEmail {
		t.Fatalf("outbox destination=%q err=%v", event.Payload.Destination, err)
	}

	badCode := dto
	badCode.CodePlaintext = "wrong"
	if _, err := f.auth.CompleteAuthentication(f.ctx, badCode); !errors.Is(err, service.ErrOtpCodeInvalid) {
		t.Fatalf("malformed OTP: %v", err)
	}
	challenge, err := f.store.ChallengeRepository().GetByChallengeId(f.ctx, dto.ChallengeID)
	if err != nil || challenge.FailedAttempts != 0 {
		t.Fatalf("malformed OTP changed attempts: challenge=%+v err=%v", challenge, err)
	}
	if _, err := f.auth.CompleteAuthentication(f.ctx, dto); err != nil {
		t.Fatalf("valid OTP after malformed input: %v", err)
	}
	f.assertState(t, canonicalEmail, dto, authenticationState{identities: 1, sessions: 1})

	verification := f.newChallenge("verify@example.com")
	f.saveChallenge(t, verification)
	mixedCase := "  Verify@Example.COM  "
	verified, err := f.service.Consume(f.ctx, domain.ConsumeChallengeDTO{
		ChallengeID: verification.ID, Code: testCode,
		ExpectedPurpose: domain.CodePurposeAuthentication, ExpectedDestination: &mixedCase,
	})
	if err != nil || verified.Destination != verification.Destination {
		t.Fatalf("normalized expected destination: verified=%+v err=%v", verified, err)
	}
}

func (f *authenticationFixture) testWrongAttempts(t *testing.T) {
	const email = "wrong@example.com"
	dto := f.issue(t, email)
	wrongCode := "000000"
	if dto.CodePlaintext == wrongCode {
		wrongCode = "111111"
	}
	dto.CodePlaintext = wrongCode
	for attempt := 1; attempt <= f.limits.MaxAttemptsPerChallenge; attempt++ {
		_, err := f.auth.CompleteAuthentication(f.ctx, dto)
		if attempt == f.limits.MaxAttemptsPerChallenge {
			if !errors.Is(err, service.ErrOtpRateLimited) {
				t.Fatalf("last attempt: %v", err)
			}
			f.assertState(t, email, dto, authenticationState{})
			continue
		}
		if !errors.Is(err, service.ErrOtpCodeInvalid) {
			t.Fatalf("attempt %d: %v", attempt, err)
		}
		challenge, err := f.store.ChallengeRepository().GetByChallengeId(f.ctx, dto.ChallengeID)
		if err != nil || challenge.FailedAttempts != attempt {
			t.Fatalf("attempt %d: persisted failures = %d, err = %v", attempt, challenge.FailedAttempts, err)
		}
		f.assertState(t, email, dto, authenticationState{challenges: 1, outbox: 1})
	}
}

func (f *authenticationFixture) testConcurrentCompletion(t *testing.T) {
	const email = "concurrent@example.com"
	dto := f.issue(t, email)
	results := make(chan error, 2)
	for range 2 {
		go func() {
			_, err := f.auth.CompleteAuthentication(f.ctx, dto)
			results <- err
		}()
	}
	successes := 0
	for range 2 {
		if err := <-results; err == nil {
			successes++
		} else if !errors.Is(err, service.ErrOtpCodeInvalid) {
			t.Fatalf("unexpected completion error: %v", err)
		}
	}
	if successes != 1 {
		t.Fatalf("successful completions = %d, expected one success", successes)
	}
	f.assertState(t, email, dto, authenticationState{identities: 1, sessions: 1})
}

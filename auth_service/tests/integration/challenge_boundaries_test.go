package integration_test

import (
	"errors"
	"testing"

	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
	"github.com/mikhaeris/sky-bank/auth_service/internal/service"
)

func TestChallengeRejectsExpiredCode(t *testing.T) {
	env := newOTPTestEnv(t)
	challenge := env.newChallenge("expired@example.invalid")
	env.saveChallenge(t, challenge)
	if _, err := env.pool.Exec(env.ctx,
		"UPDATE challenges SET expires_at = now() - interval '1 second' WHERE id = $1", challenge.ID,
	); err != nil {
		t.Fatal(err)
	}
	if err := env.consume(challenge.ID, testCode); !errors.Is(err, service.ErrOtpCodeInvalid) {
		t.Fatalf("expired code was accepted: %v", err)
	}
}

func TestChallengePurposeAndDestinationAreBoundToCode(t *testing.T) {
	env := newOTPTestEnv(t)

	t.Run("wrong purpose leaves code available", func(t *testing.T) {
		challenge := env.newChallenge("purpose@example.invalid")
		env.saveChallenge(t, challenge)
		_, err := env.service.Consume(env.ctx, domain.ConsumeChallengeDTO{
			ChallengeID: challenge.ID, Code: testCode,
			ExpectedPurpose: domain.CodePurposeEmailVerification,
		})
		if !errors.Is(err, service.ErrPurposeInvalid) {
			t.Fatalf("wrong purpose: %v", err)
		}
		if err := env.consume(challenge.ID, testCode); err != nil {
			t.Fatalf("correct purpose after rejection: %v", err)
		}
	})

	t.Run("wrong destination leaves code available", func(t *testing.T) {
		challenge := env.newChallenge("destination@example.invalid")
		env.saveChallenge(t, challenge)
		other := "other@example.invalid"
		_, err := env.service.Consume(env.ctx, domain.ConsumeChallengeDTO{
			ChallengeID: challenge.ID, Code: testCode,
			ExpectedPurpose: domain.CodePurposeAuthentication, ExpectedDestination: &other,
		})
		if !errors.Is(err, service.ErrDestinationInvalid) {
			t.Fatalf("wrong destination: %v", err)
		}
		if err := env.consume(challenge.ID, testCode); err != nil {
			t.Fatalf("correct destination after rejection: %v", err)
		}
	})
}

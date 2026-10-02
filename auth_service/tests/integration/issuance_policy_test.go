package integration_test

import (
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
	"github.com/mikhaeris/sky-bank/auth_service/internal/service"
)

func TestIssuancePolicyIntegration(t *testing.T) {
	env := newOTPTestEnv(t)
	dto := domain.IssueChallengeDTO{
		Destination: "issuance-policy@example.invalid", Channel: domain.OtpChannelEmail,
		Purpose: domain.CodePurposeAuthentication,
	}
	first, err := env.service.Issue(env.ctx, dto)
	if err != nil {
		t.Fatal(err)
	}
	var last, updated time.Time
	var count int
	read := func() {
		t.Helper()
		if err := env.pool.QueryRow(env.ctx, `SELECT last_issued_at, updated_at, issue_count
			FROM otp_limits WHERE destination = $1`, dto.Destination).Scan(&last, &updated, &count); err != nil {
			t.Fatal(err)
		}
	}
	read()
	initialLast, initialUpdated := last, updated
	for range 3 {
		_, err := env.service.Issue(env.ctx, dto)
		var retry *service.IssueRateLimitError
		if !errors.Is(err, service.ErrOtpRateLimited) || !errors.As(err, &retry) || !retry.RetryAt().Equal(initialLast.Add(15*time.Second)) {
			t.Fatalf("rejection lost retry deadline: %v", err)
		}
		read()
		if count != 1 || !last.Equal(initialLast) || !updated.Equal(initialUpdated) {
			t.Fatalf("rejection changed limits: count=%d last=%v updated=%v", count, last, updated)
		}
	}
	var current uuid.UUID
	var pending int
	if err := env.pool.QueryRow(env.ctx, `SELECT id, (SELECT count(*) FROM otp_outbox WHERE challenge_id = challenges.id)
		FROM challenges WHERE destination = $1`, dto.Destination).Scan(&current, &pending); err != nil {
		t.Fatal(err)
	}
	if current != first || pending != 1 {
		t.Fatal("rejected issuance replaced the challenge or event")
	}
	for i, wait := range []int{15, 15, 15, 15, 30, 60, 120, 240, 300, 300, 300} {
		if _, err := env.pool.Exec(env.ctx, `UPDATE otp_limits
			SET last_issued_at = now() - $2 * interval '1 second'
			WHERE destination = $1`, dto.Destination, wait); err != nil {
			t.Fatal(err)
		}
		if _, err := env.service.Issue(env.ctx, dto); err != nil {
			t.Fatalf("issue %d: %v", i+2, err)
		}
		read()
		if count != i+2 {
			t.Fatalf("series reset despite recent issuance: count=%d want=%d", count, i+2)
		}
	}
	if _, err := env.pool.Exec(env.ctx, `UPDATE otp_limits SET last_issued_at = now() - interval '30 minutes'
		WHERE destination = $1`, dto.Destination); err != nil {
		t.Fatal(err)
	}
	if _, err := env.service.Issue(env.ctx, dto); err != nil {
		t.Fatal(err)
	}
	read()
	if count != 1 {
		t.Fatalf("idle reset: count=%d", count)
	}

	challenge := env.newChallenge("verify-during-issue-cooldown@example.invalid")
	env.saveChallenge(t, challenge)
	if _, err := env.pool.Exec(env.ctx, `INSERT INTO otp_limits
		(destination, channel, purpose, last_issued_at, issue_count)
		VALUES ($1, $2, $3, now(), 100)`, challenge.Destination, challenge.Channel, challenge.Purpose); err != nil {
		t.Fatal(err)
	}
	if err := env.consume(challenge.ID, testCode); err != nil {
		t.Fatalf("issuance cooldown prevented valid code verification: %v", err)
	}
}

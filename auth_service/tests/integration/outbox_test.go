package integration_test

import (
	"errors"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
	"github.com/mikhaeris/sky-bank/auth_service/internal/repository"
	"github.com/mikhaeris/sky-bank/auth_service/internal/service"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestOTPOutboxIntegration(t *testing.T) {
	env := newOTPTestEnv(t)
	ctx, pool := env.ctx, env.pool
	tracer := noop.NewTracerProvider().Tracer("test")
	repo := env.store.OTPOutboxRepository()
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	reset := func() { exec("TRUNCATE otp_outbox, challenges") }
	insert := func() domain.OTPOutbox {
		t.Helper()
		event := domain.OTPOutbox{
			EventID: uuid.New(), ChallengeID: uuid.New(), EncryptedEvent: []byte("ciphertext"),
			TraceParent: "parent", TraceState: "state", ExpiresAt: time.Now().Add(time.Hour),
		}
		exec(`INSERT INTO challenges (id, destination, channel, purpose, code_hash, expires_at)
		      VALUES ($1, $2, 'email', 'authentication', $3, $4)`,
			event.ChallengeID, event.ChallengeID.String(), []byte("hash"), event.ExpiresAt)
		if err := repo.Insert(ctx, event); err != nil {
			t.Fatal(err)
		}
		return event
	}
	claim := func(limit int) []domain.OTPOutbox {
		t.Helper()
		events, err := repo.Claim(ctx, limit, time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		return events
	}

	for _, consumed := range []bool{true, false} {
		name := "consumed challenge"
		if !consumed {
			name = "exhausted attempts"
		}
		t.Run(name, func(t *testing.T) {
			reset()
			challenge := env.newChallenge(uuid.New().String() + "@example.invalid")
			env.saveChallenge(t, challenge)
			event := domain.OTPOutbox{
				EventID: uuid.New(), ChallengeID: challenge.ID,
				EncryptedEvent: []byte("ciphertext"), ExpiresAt: challenge.ExpiresAt,
			}
			if err := repo.Insert(ctx, event); err != nil {
				t.Fatal(err)
			}
			if consumed {
				if err := env.consume(challenge.ID, testCode); err != nil {
					t.Fatal(err)
				}
			} else {
				for attempt := range env.limits.MaxAttemptsPerChallenge {
					want := service.ErrOtpCodeInvalid
					if attempt+1 == env.limits.MaxAttemptsPerChallenge {
						want = service.ErrOtpRateLimited
					}
					if err := env.consume(challenge.ID, "000000"); !errors.Is(err, want) {
						t.Fatalf("wrong code: got %v, want %v", err, want)
					}
					if attempt+1 < env.limits.MaxAttemptsPerChallenge {
						var pending bool
						if err := pool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM otp_outbox WHERE event_id = $1)", event.EventID).Scan(&pending); err != nil || !pending {
							t.Fatalf("outbox removed before final attempt: pending=%v err=%v", pending, err)
						}
					}
				}
			}
			var pending, challengeExists bool
			if err := pool.QueryRow(ctx, `SELECT
				EXISTS (SELECT 1 FROM otp_outbox WHERE event_id = $1),
				EXISTS (SELECT 1 FROM challenges WHERE id = $2)`, event.EventID, challenge.ID).Scan(&pending, &challengeExists); err != nil {
				t.Fatal(err)
			}
			if pending || challengeExists {
				t.Fatalf("Consume left records: outbox=%v challenge=%v", pending, challengeExists)
			}
			if events := claim(1); len(events) != 0 {
				t.Fatal("event for a deleted challenge was claimed")
			}
		})
	}

	t.Run("outbox deletion failure rolls back Consume", func(t *testing.T) {
		exec(`CREATE FUNCTION reject_outbox_delete() RETURNS trigger LANGUAGE plpgsql AS $$
			BEGIN RAISE EXCEPTION 'test outbox deletion failure'; END $$`)
		exec(`CREATE TRIGGER reject_outbox_delete BEFORE DELETE ON otp_outbox
			FOR EACH ROW EXECUTE FUNCTION reject_outbox_delete()`)
		t.Cleanup(func() {
			exec("DROP TRIGGER reject_outbox_delete ON otp_outbox")
			exec("DROP FUNCTION reject_outbox_delete()")
		})
		for _, consumed := range []bool{true, false} {
			name := "correct code"
			if !consumed {
				name = "last failed attempt"
			}
			t.Run(name, func(t *testing.T) {
				reset()
				challenge := env.newChallenge(uuid.New().String() + "@example.invalid")
				env.saveChallenge(t, challenge)
				code, attempts := testCode, 0
				if !consumed {
					code, attempts = "000000", env.limits.MaxAttemptsPerChallenge-1
					exec("UPDATE challenges SET failed_attempts = $2 WHERE id = $1", challenge.ID, attempts)
				}
				event := domain.OTPOutbox{
					EventID: uuid.New(), ChallengeID: challenge.ID,
					EncryptedEvent: []byte("ciphertext"), ExpiresAt: challenge.ExpiresAt,
				}
				if err := repo.Insert(ctx, event); err != nil {
					t.Fatal(err)
				}
				err := env.consume(challenge.ID, code)
				var pgErr *pgconn.PgError
				if !errors.As(err, &pgErr) || pgErr.Message != "test outbox deletion failure" {
					t.Fatalf("expected injected deletion failure, got %v", err)
				}
				var gotAttempts int
				var pending, limitExists bool
				if err := pool.QueryRow(ctx, `SELECT
					(SELECT failed_attempts FROM challenges WHERE id = $1),
					EXISTS (SELECT 1 FROM otp_outbox WHERE event_id = $2),
					EXISTS (SELECT 1 FROM otp_limits WHERE destination = $3)`,
					challenge.ID, event.EventID, challenge.Destination).Scan(&gotAttempts, &pending, &limitExists); err != nil {
					t.Fatal(err)
				}
				if gotAttempts != attempts || !pending || limitExists {
					t.Fatalf("partial commit: attempts=%d pending=%v limitExists=%v", gotAttempts, pending, limitExists)
				}
			})
		}
	})

	t.Run("replacement removes old event", func(t *testing.T) {
		reset()
		dto := domain.IssueChallengeDTO{
			Destination: uuid.New().String() + "@example.invalid",
			Channel:     domain.OtpChannelEmail, Purpose: domain.CodePurposeAuthentication,
		}
		first, err := env.service.Issue(ctx, dto)
		if err != nil {
			t.Fatal(err)
		}
		exec("UPDATE otp_limits SET last_issued_at = now() - interval '1 minute' WHERE destination = $1", dto.Destination)
		second, err := env.service.Issue(ctx, dto)
		if err != nil {
			t.Fatal(err)
		}
		var oldExists bool
		if err := pool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM otp_outbox WHERE challenge_id = $1)", first).Scan(&oldExists); err != nil {
			t.Fatal(err)
		}
		if events := claim(10); oldExists || len(events) != 1 || events[0].ChallengeID != second {
			t.Fatalf("incorrect replacement: oldExists=%v events=%+v", oldExists, events)
		}
	})

	t.Run("eligibility and cleanup", func(t *testing.T) {
		reset()
		active := insert()
		expired := insert()
		exec("UPDATE otp_outbox SET expires_at = now() - interval '1 hour' WHERE event_id = $1", expired.EventID)
		events := claim(10)
		if len(events) != 1 || events[0].EventID != active.EventID || events[0].LeasedUntil.IsZero() ||
			events[0].TraceParent != active.TraceParent || events[0].TraceState != active.TraceState ||
			string(events[0].EncryptedEvent) != string(active.EncryptedEvent) {
			t.Fatalf("unexpected claimed events: %+v", events)
		}
		if events := claim(10); len(events) != 0 {
			t.Fatal("active lease was claimed again")
		}
		if count, err := repo.DeleteExpired(ctx); err != nil || count != 1 {
			t.Fatalf("cleanup: count=%d err=%v", count, err)
		}
	})

	t.Run("reclaim and stale acknowledgement", func(t *testing.T) {
		reset()
		event := insert()
		exec("UPDATE otp_outbox SET trace_parent = NULL, trace_state = NULL WHERE event_id = $1", event.EventID)
		first := claim(1)
		if len(first) != 1 || first[0].TraceParent != "" || first[0].TraceState != "" {
			t.Fatalf("unexpected first claim: %+v", first)
		}
		exec("UPDATE otp_outbox SET leased_until = now() - interval '1 hour' WHERE event_id = $1", event.EventID)
		second := claim(1)
		if len(second) != 1 || second[0].EventID != event.EventID {
			t.Fatal("expired lease was not reclaimed")
		}
		if deleted, err := repo.DeleteClaimed(ctx, event.EventID, first[0].LeasedUntil); err != nil || deleted {
			t.Fatalf("stale lease deleted event: deleted=%v err=%v", deleted, err)
		}
		if deleted, err := repo.DeleteClaimed(ctx, event.EventID, second[0].LeasedUntil); err != nil || !deleted {
			t.Fatalf("current lease acknowledgement: deleted=%v err=%v", deleted, err)
		}
		if deleted, err := repo.DeleteClaimed(ctx, event.EventID, second[0].LeasedUntil); err != nil || deleted {
			t.Fatalf("repeated acknowledgement: deleted=%v err=%v", deleted, err)
		}
	})

	t.Run("deleted claimed event is no longer publishable", func(t *testing.T) {
		reset()
		event := insert()
		leased := claim(1)
		if len(leased) != 1 {
			t.Fatalf("claimed %d events, want 1", len(leased))
		}
		current, err := repo.IsClaimed(ctx, event.EventID, leased[0].LeasedUntil)
		if err != nil || !current {
			t.Fatalf("current lease: current=%v err=%v", current, err)
		}
		if err := repo.DeleteByChallengeID(ctx, event.ChallengeID); err != nil {
			t.Fatal(err)
		}
		current, err = repo.IsClaimed(ctx, event.EventID, leased[0].LeasedUntil)
		if err != nil || current {
			t.Fatalf("deleted lease: current=%v err=%v", current, err)
		}
	})

	t.Run("skip locked and rollback", func(t *testing.T) {
		reset()
		insert()
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)
		leased, err := repository.NewOTPOutboxRepository(tracer, tx).Claim(ctx, 1, time.Minute)
		if err != nil || len(leased) != 1 {
			t.Fatalf("transaction claim: count=%d err=%v", len(leased), err)
		}
		if events := claim(1); len(events) != 0 {
			t.Fatal("locked event was claimed")
		}
		if err := tx.Rollback(ctx); err != nil {
			t.Fatal(err)
		}
		if events := claim(1); len(events) != 1 {
			t.Fatal("rollback did not release event")
		}
	})

	t.Run("concurrent workers", func(t *testing.T) {
		reset()
		for range 24 {
			insert()
		}
		var wg sync.WaitGroup
		start := make(chan struct{})
		results := make(chan []domain.OTPOutbox, 8)
		for range 8 {
			wg.Go(func() {
				<-start
				events, err := repo.Claim(ctx, 4, time.Minute)
				if err != nil {
					t.Error(err)
				}
				results <- events
			})
		}
		close(start)
		wg.Wait()
		close(results)
		seen := make(map[uuid.UUID]bool)
		for events := range results {
			if len(events) > 4 {
				t.Fatal("batch limit exceeded")
			}
			for _, event := range events {
				if seen[event.EventID] {
					t.Fatal("event leased to multiple workers")
				}
				seen[event.EventID] = true
			}
		}
		if len(seen) != 24 {
			t.Fatalf("claimed %d of 24 events", len(seen))
		}
	})
}

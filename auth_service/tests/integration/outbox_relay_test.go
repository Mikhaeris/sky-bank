package integration_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"
	"uuid"

	"github.com/mikhaeris/sky-bank/auth_service/internal/app/workers"
	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
	"github.com/mikhaeris/sky-bank/auth_service/internal/service"
	ke "github.com/mikhaeris/sky-bank/pkg/kafkaevents"
	"github.com/mikhaeris/sky-bank/pkg/kafkaevents/constant"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

type relayAttempt struct {
	topic   string
	event   ke.Event[ke.OtpPayload]
	traceID trace.TraceID
}

type relayTestProducer struct {
	attempts  chan relayAttempt
	failFirst bool
}

func (p *relayTestProducer) Produce(ctx context.Context, topic string, event ke.BaseEvent) error {
	otpEvent, ok := event.(ke.Event[ke.OtpPayload])
	if !ok {
		return errors.New("unexpected event type")
	}
	attempt := relayAttempt{
		topic: topic, event: otpEvent, traceID: trace.SpanContextFromContext(ctx).TraceID(),
	}
	select {
	case p.attempts <- attempt:
	case <-ctx.Done():
		return ctx.Err()
	}
	if p.failFirst {
		p.failFirst = false
		return errors.New("temporary Kafka failure")
	}
	return nil
}

func TestOTPOutboxRelayPublishesAndRetries(t *testing.T) {
	env := newOTPTestEnv(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	repo := env.store.OTPOutboxRepository()
	tracer := noop.NewTracerProvider().Tracer("test")
	producer := &relayTestProducer{attempts: make(chan relayAttempt, 2), failFirst: true}

	previousPropagator := otel.GetTextMapPropagator()
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() { otel.SetTextMapPropagator(previousPropagator) })
	const traceParent = "00-11111111111111111111111111111111-2222222222222222-01"
	challenge := env.newChallenge(uuid.New().String() + "@example.invalid")
	env.saveChallenge(t, challenge)
	event := ke.Event[ke.OtpPayload]{
		Metadata: ke.Metadata{EventID: uuid.New(), AggregateID: challenge.ID},
		Payload: ke.OtpPayload{
			Destination: challenge.Destination,
			Data:        map[string]any{"otpCode": testCode},
			ExpiredAt:   &challenge.ExpiresAt,
		},
	}
	encrypted, err := env.cipher.Encrypt(event)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Insert(env.ctx, domain.OTPOutbox{
		EventID: event.Metadata.EventID, ChallengeID: challenge.ID,
		EncryptedEvent: encrypted, TraceParent: traceParent, ExpiresAt: challenge.ExpiresAt,
	}); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(env.ctx)
	done := make(chan struct{})
	relay := workers.NewOTPOutboxRelay(logger, tracer, repo, env.cipher, producer)
	go func() {
		defer close(done)
		relay.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("relay did not stop after cancellation")
		}
	})

	awaitAttempt := func() relayAttempt {
		t.Helper()
		select {
		case attempt := <-producer.attempts:
			if attempt.topic != constant.TopicOTPRequested || attempt.event.Metadata.EventID != event.Metadata.EventID ||
				attempt.event.Metadata.AggregateID != challenge.ID ||
				attempt.event.Payload.Data["otpCode"] != testCode ||
				attempt.traceID.String() != "11111111111111111111111111111111" {
				t.Fatalf("unexpected publication: %+v", attempt)
			}
			return attempt
		case <-time.After(2 * time.Second):
			t.Fatal("relay did not publish the outbox event")
			return relayAttempt{}
		}
	}
	awaitAttempt()

	var pending bool
	if err := env.pool.QueryRow(env.ctx,
		"SELECT EXISTS (SELECT 1 FROM otp_outbox WHERE event_id = $1)", event.Metadata.EventID,
	).Scan(&pending); err != nil || !pending {
		t.Fatalf("failed publication lost event: pending=%v err=%v", pending, err)
	}
	if _, err := env.pool.Exec(env.ctx,
		"UPDATE otp_outbox SET leased_until = now() - interval '1 second' WHERE event_id = $1",
		event.Metadata.EventID,
	); err != nil {
		t.Fatal(err)
	}
	relay.Wake()
	awaitAttempt()

	deadline := time.Now().Add(2 * time.Second)
	for {
		if err := env.pool.QueryRow(env.ctx,
			"SELECT EXISTS (SELECT 1 FROM otp_outbox WHERE event_id = $1)", event.Metadata.EventID,
		).Scan(&pending); err != nil {
			t.Fatal(err)
		}
		if !pending {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("successfully published event was not acknowledged")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestIssueWakesOutboxOnlyAfterCommit(t *testing.T) {
	env := newOTPTestEnv(t)
	tracer := noop.NewTracerProvider().Tracer("test")
	destination := uuid.New().String() + "@example.invalid"
	wakeCount := 0
	wake := func() {
		wakeCount++
		var count int
		err := env.pool.QueryRow(env.ctx, `SELECT count(*) FROM otp_outbox o
			JOIN challenges c ON c.id = o.challenge_id WHERE c.destination = $1`, destination).Scan(&count)
		if err != nil || count != 1 {
			t.Fatalf("outbox row is not committed at wake: count=%d err=%v", count, err)
		}
	}
	challengeService := service.NewChallengeService(tracer, env.hasher, env.cipher, wake, env.store, env.limits)
	dto := domain.IssueChallengeDTO{
		Destination: destination,
		Channel:     domain.OtpChannelEmail,
		Purpose:     domain.CodePurposeAuthentication,
	}
	if _, err := challengeService.Issue(env.ctx, dto); err != nil {
		t.Fatal(err)
	}
	if wakeCount != 1 {
		t.Fatalf("wake count after commit: got %d, want 1", wakeCount)
	}
	if _, err := challengeService.Issue(env.ctx, dto); !errors.Is(err, service.ErrOtpRateLimited) {
		t.Fatalf("second issue: got %v, want rate limit", err)
	}
	if wakeCount != 1 {
		t.Fatalf("failed issue woke relay: wake count=%d", wakeCount)
	}
}

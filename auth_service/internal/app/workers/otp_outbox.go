package workers

import (
	"context"
	"fmt"
	"log/slog"
	"time"
	"uuid"

	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
	"github.com/mikhaeris/sky-bank/auth_service/internal/lib/hasher"
	ke "github.com/mikhaeris/sky-bank/pkg/kafkaevents"
	"github.com/mikhaeris/sky-bank/pkg/kafkaevents/constant"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	outboxPollInterval    = 5 * time.Second
	outboxCleanupInterval = time.Minute
	outboxLeaseDuration   = 10 * time.Second
	outboxSendTimeout     = 5 * time.Second
)

type otpOutboxStore interface {
	Claim(context.Context, int, time.Duration) ([]domain.OTPOutbox, error)
	IsClaimed(context.Context, uuid.UUID, time.Time) (bool, error)
	DeleteClaimed(context.Context, uuid.UUID, time.Time) (bool, error)
	DeleteExpired(context.Context) (int64, error)
}

type otpEventProducer interface {
	Produce(context.Context, string, ke.BaseEvent) error
}

type OTPOutboxRelay struct {
	logger   *slog.Logger
	tracer   trace.Tracer
	store    otpOutboxStore
	cipher   *hasher.EventCipher
	producer otpEventProducer
	wake     chan struct{}
}

func NewOTPOutboxRelay(
	logger *slog.Logger,
	tracer trace.Tracer,
	store otpOutboxStore,
	cipher *hasher.EventCipher,
	producer otpEventProducer,
) *OTPOutboxRelay {
	return &OTPOutboxRelay{
		logger: logger, tracer: tracer, store: store, cipher: cipher, producer: producer,
		wake: make(chan struct{}, 1),
	}
}

func (r *OTPOutboxRelay) Wake() {
	select {
	case r.wake <- struct{}{}:
	default:
	}
}

func (r *OTPOutboxRelay) Run(ctx context.Context) {
	r.cleanup(ctx)
	r.drain(ctx)

	poll := time.NewTicker(outboxPollInterval)
	defer poll.Stop()
	cleanup := time.NewTicker(outboxCleanupInterval)
	defer cleanup.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.wake:
			r.drain(ctx)
		case <-poll.C:
			r.drain(ctx)
		case <-cleanup.C:
			r.cleanup(ctx)
		}
	}
}

func (r *OTPOutboxRelay) drain(ctx context.Context) {
	for ctx.Err() == nil {
		events, err := r.store.Claim(ctx, 1, outboxLeaseDuration)
		if err != nil {
			if ctx.Err() == nil {
				r.logger.Error("claim OTP outbox", "error", err)
			}
			return
		}
		if len(events) == 0 {
			return
		}
		event := events[0]
		if err := r.publish(ctx, event); err != nil && ctx.Err() == nil {
			r.logger.Error("publish OTP outbox", "event_id", event.EventID, "error", err)
		}
	}
}

func (r *OTPOutboxRelay) publish(ctx context.Context, row domain.OTPOutbox) (err error) {
	carrier := propagation.MapCarrier{}
	if row.TraceParent != "" {
		carrier.Set("traceparent", row.TraceParent)
	}
	if row.TraceState != "" {
		carrier.Set("tracestate", row.TraceState)
	}
	ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)
	ctx, span := r.tracer.Start(ctx, "otp_outbox.publish")
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	event, err := r.cipher.Decrypt(row.EncryptedEvent, row.EventID, row.ChallengeID)
	if err != nil {
		return fmt.Errorf("decrypt event: %w", err)
	}

	current, err := r.store.IsClaimed(ctx, row.EventID, row.LeasedUntil)
	if err != nil {
		return fmt.Errorf("check event before publish: %w", err)
	}
	if !current {
		return nil
	}

	sendCtx, cancel := context.WithTimeout(ctx, outboxSendTimeout)
	defer cancel()
	if err := r.producer.Produce(sendCtx, constant.TopicOTPRequested, event); err != nil {
		return fmt.Errorf("send event to Kafka: %w", err)
	}

	deleted, err := r.store.DeleteClaimed(ctx, row.EventID, row.LeasedUntil)
	if err != nil {
		return fmt.Errorf("acknowledge published event: %w", err)
	}
	if !deleted {
		return fmt.Errorf("event %s was published but its lease is no longer current", row.EventID)
	}
	return nil
}

func (r *OTPOutboxRelay) cleanup(ctx context.Context) {
	count, err := r.store.DeleteExpired(ctx)
	if err != nil {
		if ctx.Err() == nil {
			r.logger.Error("clean expired OTP outbox", "error", err)
		}
		return
	}
	if count > 0 {
		r.logger.Info("cleaned expired OTP outbox", "count", count)
	}
}

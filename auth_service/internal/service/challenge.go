package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"
	"uuid"

	"github.com/mikhaeris/sky-bank/auth_service/internal/config"
	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
	"github.com/mikhaeris/sky-bank/auth_service/internal/lib/hasher"
	"github.com/mikhaeris/sky-bank/auth_service/internal/repository"
	ke "github.com/mikhaeris/sky-bank/pkg/kafkaevents"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	minOTPLimitRetention = 24 * time.Hour
	cooldownMultiplier   = 2
	otpCodeUpperBound    = 1_000_000
	otpCodeFormat        = "%06d"
)

type ChallengeService struct {
	tracer     trace.Tracer
	codeHash   *hasher.CodeHasher
	outboxHash *hasher.EventCipher
	wakeOutbox func()
	store      repository.DataStore
	limits     config.OtpLimits
}

func NewChallengeService(
	tracer trace.Tracer,
	codeHash *hasher.CodeHasher,
	outboxHash *hasher.EventCipher,
	wakeOutbox func(),
	store repository.DataStore,
	limits config.OtpLimits,
) *ChallengeService {
	return &ChallengeService{
		tracer:     tracer,
		codeHash:   codeHash,
		outboxHash: outboxHash,
		wakeOutbox: wakeOutbox,
		store:      store,
		limits:     limits,
	}
}

func NormalizeDestination(destination string, channel domain.OtpChannel) string {
	destination = strings.TrimSpace(destination)
	if channel == domain.OtpChannelEmail {
		return strings.ToLower(destination)
	}
	return destination
}

func NextIssueState(now time.Time, limit domain.OTPLimitState, limits config.OtpLimits) (domain.OTPLimitState, error) {
	count := 0
	if limit.LastIssuedAt != nil && now.Before(limit.LastIssuedAt.Add(limits.IssueResetAfter)) {
		count = limit.IssueCount
	}
	retryAt := now
	if limit.LastIssuedAt != nil {
		if next := limit.LastIssuedAt.Add(issueCooldown(count, limits)); next.After(retryAt) {
			retryAt = next
		}
	}
	if now.Before(retryAt) {
		return domain.OTPLimitState{}, &IssueRateLimitError{AvailableAt: retryAt}
	}
	limit.LastIssuedAt = &now
	limit.IssueCount = count + 1
	limit.UpdatedAt = now
	return limit, nil
}

func issueCooldown(issuedInSeries int, limits config.OtpLimits) time.Duration {
	cooldown := limits.InitialCooldown
	for issued := limits.InitialIssues; issued <= issuedInSeries && cooldown < limits.MaxCooldown; issued++ {
		if cooldown > limits.MaxCooldown/cooldownMultiplier {
			return limits.MaxCooldown
		}
		cooldown *= cooldownMultiplier
	}
	return cooldown
}

func (c *ChallengeService) Cleanup(ctx context.Context) (int64, int64, error) {
	now := time.Now().UTC()
	expiredChallenges, err := c.store.ChallengeRepository().DeleteExpired(ctx, now)
	if err != nil {
		return 0, 0, fmt.Errorf("delete expired challenges: %w", err)
	}

	retention := max(minOTPLimitRetention, c.limits.IssueResetAfter, c.limits.MaxCooldown)
	inactiveLimits, err := c.store.OTPLimitRepository().DeleteInactive(ctx, now.Add(-retention))
	if err != nil {
		return expiredChallenges, 0, fmt.Errorf("delete inactive otp limits: %w", err)
	}
	return expiredChallenges, inactiveLimits, nil
}

func (c *ChallengeService) generateChallenge(
	ctx context.Context,
	hasher *hasher.CodeHasher,
	destination string,
	channel domain.OtpChannel,
	purpose domain.CodePurpose,
	ttl time.Duration,
) (*domain.Challenge, string, error) {
	_, span := c.tracer.Start(ctx, "challenge.generate")
	defer span.End()

	n, err := rand.Int(rand.Reader, big.NewInt(otpCodeUpperBound))
	if err != nil {
		return nil, "", fmt.Errorf("generate code: %w", err)
	}

	plaintext := fmt.Sprintf(otpCodeFormat, n.Int64())

	return &domain.Challenge{
		ID:          uuid.New(),
		Destination: destination,
		Channel:     channel,
		Purpose:     purpose,
		CodeHash:    hasher.Hash(plaintext),
		ExpiresAt:   time.Now().Add(ttl),
	}, plaintext, nil
}

func (c *ChallengeService) Issue(ctx context.Context, dto domain.IssueChallengeDTO) (uuid.UUID, error) {
	ctx, span := c.tracer.Start(ctx, "challenge.issue")
	defer span.End()
	dto.Destination = NormalizeDestination(dto.Destination, dto.Channel)

	challenge, codePlaintext, err := c.generateChallenge(
		ctx,
		c.codeHash,
		dto.Destination,
		domain.OtpChannel(dto.Channel),
		domain.CodePurpose(dto.Purpose),
		domain.CodeTTLAuthentication,
	)
	if err != nil {
		return uuid.Nil(), fmt.Errorf("issue challenge: %w", err)
	}

	data := map[string]any{
		"otpCode": codePlaintext,
	}

	event := ke.Event[ke.OtpPayload]{
		Metadata: ke.Metadata{
			EventID:     uuid.New(),
			AggregateID: challenge.ID,
			EventType:   "otp.requested",
			OccurredAt:  time.Now().UTC(),
			Source:      "auth_service",
		},
		Payload: ke.OtpPayload{
			IdentityID:       uuid.Nil(),
			Destination:      dto.Destination,
			NotificationType: ke.NotificationTypeEmail,
			Data:             data,
			ExpiredAt:        &challenge.ExpiresAt,
		},
	}

	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)

	traceParent := carrier.Get("traceparent")
	traceState := carrier.Get("tracestate")

	key := challenge.Destination
	err = c.store.Atomic(ctx, func(ctx context.Context, tx repository.TxStore) error {
		limitsRepo := tx.OTPLimitRepository()
		otpOutboxRepo := tx.OTPOutboxRepository()
		challengeRepo := tx.ChallengeRepository()

		limit, err := limitsRepo.Lock(ctx, key, challenge.Channel, challenge.Purpose)
		if err != nil {
			return err
		}

		now := time.Now().UTC()
		next, err := NextIssueState(now, limit, c.limits)
		if err != nil {
			return err
		}

		if err := limitsRepo.RecordIssue(ctx, next); err != nil {
			return err
		}

		oldChallengeID, err := challengeRepo.Upster(ctx, challenge)
		if err != nil {
			return err
		}
		if oldChallengeID != uuid.Nil() {
			err = otpOutboxRepo.DeleteByChallengeID(ctx, oldChallengeID)
			if err != nil {
				return err
			}
		}

		encryptedEvent, err := c.outboxHash.Encrypt(event)
		if err != nil {
			return err
		}

		otpOutbox := domain.OTPOutbox{
			EventID:        event.Metadata.EventID,
			ChallengeID:    challenge.ID,
			EncryptedEvent: encryptedEvent,
			TraceParent:    traceParent,
			TraceState:     traceState,
			ExpiresAt:      challenge.ExpiresAt,
		}

		err = otpOutboxRepo.Insert(ctx, otpOutbox)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		if errors.Is(err, ErrOtpRateLimited) {
			return uuid.Nil(), err
		}
		return uuid.Nil(), fmt.Errorf("issue challenge: %w", err)
	}

	c.wakeOutbox()

	return challenge.ID, nil
}

func (c *ChallengeService) Consume(ctx context.Context, dto domain.ConsumeChallengeDTO) (domain.VerifiedChallenge, error) {
	return c.consume(ctx, dto, nil)
}

func (c *ChallengeService) consume(
	ctx context.Context,
	dto domain.ConsumeChallengeDTO,
	onVerified func(context.Context, repository.TxStore, domain.VerifiedChallenge) error,
) (domain.VerifiedChallenge, error) {
	ctx, span := c.tracer.Start(ctx, "challenge.consume")
	defer span.End()
	if !validOTPCode(dto.Code) {
		return domain.VerifiedChallenge{}, ErrOtpCodeInvalid
	}

	var verified domain.VerifiedChallenge
	var resultErr error
	err := c.store.Atomic(ctx, func(ctx context.Context, tx repository.TxStore) error {
		challenges := tx.ChallengeRepository()
		challenge, err := challenges.GetByChallengeIdForUpdate(ctx, dto.ChallengeID)
		if errors.Is(err, repository.ErrRecordNotFound) {
			return ErrOtpCodeInvalid
		}
		if err != nil {
			return err
		}

		if challenge.Purpose != dto.ExpectedPurpose {
			return ErrPurposeInvalid
		}
		if dto.ExpectedDestination != nil && challenge.Destination != NormalizeDestination(*dto.ExpectedDestination, challenge.Channel) {
			return ErrDestinationInvalid
		}

		if challenge.FailedAttempts >= c.limits.MaxAttemptsPerChallenge {
			return ErrOtpRateLimited
		}

		if !c.codeHash.Verify(dto.Code, challenge.CodeHash) {
			resultErr = ErrOtpCodeInvalid
			if challenge.FailedAttempts+1 >= c.limits.MaxAttemptsPerChallenge {
				if err := challenges.DeleteByChallengeId(ctx, challenge.ID); err != nil {
					return err
				}
				if err := tx.OTPOutboxRepository().DeleteByChallengeID(ctx, challenge.ID); err != nil {
					return err
				}
				resultErr = ErrOtpRateLimited
			} else if err := challenges.IncrementFailedAttempts(ctx, challenge.ID); err != nil {
				return err
			}
			return nil
		}

		if err := challenges.DeleteByChallengeId(ctx, challenge.ID); err != nil {
			return err
		}
		if err := tx.OTPOutboxRepository().DeleteByChallengeID(ctx, challenge.ID); err != nil {
			return err
		}
		verified = domain.VerifiedChallenge{
			Destination: challenge.Destination,
			Channel:     challenge.Channel,
			Purpose:     challenge.Purpose,
		}
		if onVerified != nil {
			return onVerified(ctx, tx, verified)
		}
		return nil
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrOtpCodeInvalid):
			return domain.VerifiedChallenge{}, ErrOtpCodeInvalid
		case errors.Is(err, ErrOtpRateLimited):
			return domain.VerifiedChallenge{}, ErrOtpRateLimited
		case errors.Is(err, ErrPurposeInvalid):
			return domain.VerifiedChallenge{}, ErrPurposeInvalid
		case errors.Is(err, ErrDestinationInvalid):
			return domain.VerifiedChallenge{}, ErrDestinationInvalid
		default:
			return domain.VerifiedChallenge{}, fmt.Errorf("consume challenge: %w", err)
		}
	}
	if resultErr != nil {
		return domain.VerifiedChallenge{}, resultErr
	}
	return verified, nil
}

func validOTPCode(code string) bool {
	if len(code) != 6 {
		return false
	}
	for i := 0; i < len(code); i++ {
		if code[i] < '0' || code[i] > '9' {
			return false
		}
	}
	return true
}

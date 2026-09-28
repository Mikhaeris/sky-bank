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
	"github.com/mikhaeris/sky-bank/auth_service/internal/lib/otp"
	"github.com/mikhaeris/sky-bank/auth_service/internal/repository"
	"github.com/mikhaeris/sky-bank/pkg/kafka"
	ke "github.com/mikhaeris/sky-bank/pkg/kafkaevents"
	"github.com/mikhaeris/sky-bank/pkg/kafkaevents/constant"
	"go.opentelemetry.io/otel/trace"
)

const (
	minOTPLimitRetention = 24 * time.Hour
	cooldownMultiplier   = 2
	otpCodeUpperBound    = 1_000_000
	otpCodeFormat        = "%06d"
)

type ChallengeService struct {
	tracer   trace.Tracer
	codeHash *otp.CodeHasher
	producer *kafka.Producer
	store    repository.DataStore
	limits   config.OtpLimits
}

func NewChallengeService(
	tracer trace.Tracer,
	codeHash *otp.CodeHasher,
	producer *kafka.Producer,
	store repository.DataStore,
	limits config.OtpLimits,
) *ChallengeService {
	return &ChallengeService{
		tracer:   tracer,
		codeHash: codeHash,
		producer: producer,
		store:    store,
		limits:   limits,
	}
}

func RateLimitDestination(destination string, channel domain.OtpChannel) string {
	destination = strings.TrimSpace(destination)
	if channel == domain.OtpChannelEmail {
		return strings.ToLower(destination)
	}
	return destination
}

func NextIssueState(now time.Time, limit domain.OTPLimitState, limits config.OtpLimits) (domain.OTPLimitState, error) {
	if limit.BlockedUntil != nil && now.Before(*limit.BlockedUntil) {
		return domain.OTPLimitState{}, ErrOtpRateLimited
	}

	windowStartedAt := now
	count := 0
	if limit.IssueWindowStartedAt != nil && now.Before(limit.IssueWindowStartedAt.Add(limits.IssueWindow)) {
		windowStartedAt = *limit.IssueWindowStartedAt
		count = limit.IssueCount
	}
	if count >= limits.MaxIssues {
		return domain.OTPLimitState{}, ErrOtpRateLimited
	}
	if limit.LastIssuedAt != nil && now.Before(limit.LastIssuedAt.Add(issueCooldown(count, limits))) {
		return domain.OTPLimitState{}, ErrOtpRateLimited
	}
	limit.LastIssuedAt = &now
	limit.IssueWindowStartedAt = &windowStartedAt
	limit.IssueCount = count + 1
	limit.UpdatedAt = now
	return limit, nil
}

func issueCooldown(issuedInWindow int, limits config.OtpLimits) time.Duration {
	cooldown := limits.InitialCooldown
	for issued := limits.FreeIssues; issued <= issuedInWindow && cooldown < limits.MaxCooldown; issued++ {
		if cooldown > limits.MaxCooldown/cooldownMultiplier {
			return limits.MaxCooldown
		}
		cooldown *= cooldownMultiplier
	}
	return cooldown
}

func NextFailureState(now time.Time, limit domain.OTPLimitState, limits config.OtpLimits) domain.OTPLimitState {
	windowStartedAt := now
	count := 0
	if limit.FailureWindowStartedAt != nil && now.Before(limit.FailureWindowStartedAt.Add(limits.FailureWindow)) {
		windowStartedAt = *limit.FailureWindowStartedAt
		count = limit.FailureCount
	}
	count++
	limit.FailureWindowStartedAt = &windowStartedAt
	limit.FailureCount = count
	limit.BlockedUntil = nil
	if count >= limits.MaxFailures {
		blockedUntil := now.Add(limits.FailureWindow)
		limit.BlockedUntil = &blockedUntil
	}
	limit.UpdatedAt = now
	return limit
}

func (c *ChallengeService) Cleanup(ctx context.Context) (int64, int64, error) {
	now := time.Now().UTC()
	expiredChallenges, err := c.store.ChallengeRepository().DeleteExpired(ctx, now)
	if err != nil {
		return 0, 0, fmt.Errorf("delete expired challenges: %w", err)
	}

	retention := max(minOTPLimitRetention, c.limits.IssueWindow, c.limits.FailureWindow, c.limits.MaxCooldown)
	inactiveLimits, err := c.store.OTPLimitRepository().DeleteInactive(ctx, now.Add(-retention), now)
	if err != nil {
		return expiredChallenges, 0, fmt.Errorf("delete inactive otp limits: %w", err)
	}
	return expiredChallenges, inactiveLimits, nil
}

func (c *ChallengeService) generateChallenge(
	ctx context.Context,
	hasher *otp.CodeHasher,
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

	key := RateLimitDestination(challenge.Destination, challenge.Channel)
	err = c.store.Atomic(ctx, func(ctx context.Context, tx repository.TxStore) error {
		limitsRepo := tx.OTPLimitRepository()

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

		return tx.ChallengeRepository().Upster(ctx, challenge)
	})
	if err != nil {
		if errors.Is(err, ErrOtpRateLimited) {
			return uuid.Nil(), ErrOtpRateLimited
		}
		return uuid.Nil(), fmt.Errorf("issue challenge: %w", err)
	}

	data := map[string]any{
		"otpCode": codePlaintext,
	}

	event := ke.Event[ke.OtpPayload]{
		Metadata: ke.Metadata{
			EventID:    uuid.New(),
			EventType:  "otp.requested",
			OccurredAt: time.Now().UTC(),
			Source:     "auth_service",
		},
		Payload: ke.OtpPayload{
			IdentityID:       uuid.Nil(),
			Destination:      dto.Destination,
			NotificationType: ke.NotificationTypeEmail,
			Data:             data,
			ExpiredAt:        &challenge.ExpiresAt,
		},
	}

	err = c.producer.Produce(ctx, constant.TopicOTPRequested, event)
	if err != nil {
		return uuid.Nil(), fmt.Errorf("publish otp request: %w", err)
	}

	return challenge.ID, nil
}

func (c *ChallengeService) Consume(ctx context.Context, dto domain.ConsumeChallengeDTO) (domain.VerifiedChallenge, error) {
	var verified domain.VerifiedChallenge
	var resultErr error
	err := c.store.Atomic(ctx, func(ctx context.Context, tx repository.TxStore) error {
		challenges := tx.ChallengeRepository()
		initial, err := challenges.GetByChallengeId(ctx, dto.ChallengeID)
		if errors.Is(err, repository.ErrRecordNotFound) {
			return ErrOtpCodeInvalid
		}
		if err != nil {
			return err
		}

		key := RateLimitDestination(initial.Destination, initial.Channel)
		limitsRepo := tx.OTPLimitRepository()
		limit, err := limitsRepo.Lock(ctx, key, initial.Channel, initial.Purpose)
		if err != nil {
			return err
		}
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
		if dto.ExpectedDestination != nil && challenge.Destination != *dto.ExpectedDestination {
			return ErrDestinationInvalid
		}

		now := time.Now().UTC()
		if limit.BlockedUntil != nil && now.Before(*limit.BlockedUntil) {
			return ErrOtpRateLimited
		}
		if challenge.FailedAttempts >= c.limits.MaxAttemptsPerChallenge {
			return ErrOtpRateLimited
		}

		if !c.codeHash.Verify(dto.Code, challenge.CodeHash) {
			next := NextFailureState(now, limit, c.limits)
			if err := limitsRepo.RecordFailure(ctx, next); err != nil {
				return err
			}
			if challenge.FailedAttempts+1 >= c.limits.MaxAttemptsPerChallenge {
				if err := challenges.DeleteByChallengeId(ctx, challenge.ID); err != nil {
					return err
				}
			} else if err := challenges.IncrementFailedAttempts(ctx, challenge.ID); err != nil {
				return err
			}
			resultErr = ErrOtpCodeInvalid
			if next.BlockedUntil != nil || challenge.FailedAttempts+1 >= c.limits.MaxAttemptsPerChallenge {
				resultErr = ErrOtpRateLimited
			}
			return nil
		}

		if err := challenges.DeleteByChallengeId(ctx, challenge.ID); err != nil {
			return err
		}
		verified = domain.VerifiedChallenge{
			Destination: challenge.Destination,
			Channel:     challenge.Channel,
			Purpose:     challenge.Purpose,
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
		case errors.Is(err, repository.ErrRecordNotFound):
			return domain.VerifiedChallenge{}, ErrOtpCodeInvalid
		default:
			return domain.VerifiedChallenge{}, fmt.Errorf("consume challenge: %w", err)
		}
	}
	if resultErr != nil {
		return domain.VerifiedChallenge{}, resultErr
	}
	return verified, nil
}

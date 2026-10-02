package service

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
	"github.com/mikhaeris/sky-bank/auth_service/internal/lib/jwt"
	"github.com/mikhaeris/sky-bank/auth_service/internal/repository"
	ke "github.com/mikhaeris/sky-bank/pkg/kafkaevents"
	"github.com/mikhaeris/sky-bank/pkg/kafkaevents/constant"
	"go.opentelemetry.io/otel/trace"
)

type NotificationProducer interface {
	Produce(context.Context, string, ke.BaseEvent) error
}

type IdentityService struct {
	tracer               trace.Tracer
	jwtKey               *jwt.Keys
	notificationProducer NotificationProducer
	challengeServ        *ChallengeService
	store                repository.DataStore
}

func NewIdentityService(
	tracer trace.Tracer,
	jwtKey *jwt.Keys,
	notificationProducer NotificationProducer,
	challengeServ *ChallengeService,
	store repository.DataStore,
) *IdentityService {
	return &IdentityService{
		tracer:               tracer,
		jwtKey:               jwtKey,
		notificationProducer: notificationProducer,
		challengeServ:        challengeServ,
		store:                store,
	}
}

func (i *IdentityService) StartAuthentication(ctx context.Context, dto domain.IdentityDTO) (uuid.UUID, error) {
	ctx, span := i.tracer.Start(ctx, "auth.start_authentication")
	defer span.End()

	tdto := domain.IssueChallengeDTO{
		Destination: dto.Email,
		Channel:     domain.OtpChannelEmail,
		Purpose:     domain.CodePurposeAuthentication,
	}

	challengeID, err := i.challengeServ.Issue(ctx, tdto)
	if err != nil {
		return uuid.Nil(), fmt.Errorf("start authentication: %w", err)
	}

	return challengeID, nil
}

func (i *IdentityService) CompleteAuthentication(ctx context.Context, dto domain.OtpDTO) (domain.Tokens, error) {
	ctx, span := i.tracer.Start(ctx, "auth.complete_authentication")
	defer span.End()

	tdto := domain.ConsumeChallengeDTO{
		ChallengeID:     dto.ChallengeID,
		Code:            dto.CodePlaintext,
		ExpectedPurpose: domain.CodePurposeAuthentication,
	}
	var identity domain.Identity
	var tokens domain.Tokens
	code, err := i.challengeServ.consume(ctx, tdto, func(ctx context.Context, tx repository.TxStore, code domain.VerifiedChallenge) error {
		identityRepo := tx.IdentityRepository()
		if err := identityRepo.InsertIfAbsent(ctx, code.Destination); err != nil {
			return fmt.Errorf("create identity: %w", err)
		}

		var txErr error
		identity, txErr = identityRepo.GetByEmail(ctx, code.Destination)
		if txErr != nil {
			return fmt.Errorf("find identity: %w", txErr)
		}

		ctx, span := i.tracer.Start(ctx, "auth.create_tokens")
		accessToken, err := i.jwtKey.CreateToken(&identity)
		if err != nil {
			return fmt.Errorf("sign access token: %w", err)
		}

		session := domain.NewSession(identity.ID, domain.RefreshTokenTTL)
		if err := tx.SessionRepository().Insert(ctx, *session); err != nil {
			return fmt.Errorf("create session: %w", err)
		}
		span.End()

		tokens = domain.Tokens{
			Access:    accessToken,
			Refresh:   session.RefreshTokenPlaintext,
			ExpiresAt: session.ExpiresAt,
		}

		return nil
	})
	if err != nil {
		return domain.Tokens{}, fmt.Errorf("complete authentication: %w", err)
	}

	go func(identity domain.Identity, code domain.VerifiedChallenge) {
		event := ke.Event[ke.NotificationPayload]{
			Metadata: ke.Metadata{
				EventID:    uuid.New(),
				EventType:  "notification.requested",
				OccurredAt: time.Now().UTC(),
				Source:     "auth_service",
			},
			Payload: ke.NotificationPayload{
				IdentityID:       identity.ID,
				Destination:      code.Destination,
				NotificationType: ke.NotificationTypeEmail,
				TemplateID:       constant.TemplateNewLogIn,
				Data:             map[string]any{},
				ExpiredAt:        nil,
			},
		}

		_ = i.notificationProducer.Produce(ctx, constant.TopicNotificationRequested, event)
	}(identity, code)

	return tokens, nil
}

func (i *IdentityService) RefreshTokens(ctx context.Context, dto domain.TokensDTO) (domain.Tokens, error) {
	ctx, span := i.tracer.Start(ctx, "auth.refresh_tokens")
	defer span.End()

	oldTokenHash := domain.HashRefreshToken(dto.Refersh)

	session, err := i.store.SessionRepository().GetByRefreshTokenHash(ctx, oldTokenHash)
	if err != nil {
		if errors.Is(err, repository.ErrRecordNotFound) {
			return domain.Tokens{}, ErrInvalidRefreshToken
		}
		return domain.Tokens{}, fmt.Errorf("refresh tokens: %w", err)
	}

	identity, err := i.store.IdentityRepository().GetByID(ctx, session.IdentityID)
	if err != nil {
		if errors.Is(err, repository.ErrRecordNotFound) {
			return domain.Tokens{}, ErrInvalidRefreshToken
		}
		return domain.Tokens{}, fmt.Errorf("refresh tokens: %w", err)
	}

	ctx, span = i.tracer.Start(ctx, "auth.create_access_tokens")
	accessToken, err := i.jwtKey.CreateToken(&identity)
	if err != nil {
		return domain.Tokens{}, fmt.Errorf("sign refreshed access token: %w", err)
	}

	session.RotateRefreshToken()
	span.End()

	err = i.store.SessionRepository().Update(ctx, session, oldTokenHash)
	if err != nil {
		if errors.Is(err, repository.ErrEditConflict) {
			return domain.Tokens{}, ErrInvalidRefreshToken
		}
		return domain.Tokens{}, fmt.Errorf("refresh tokens: %w", err)
	}

	return domain.Tokens{
		Access:    accessToken,
		Refresh:   session.RefreshTokenPlaintext,
		ExpiresAt: session.ExpiresAt,
	}, nil
}

package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
	"github.com/mikhaeris/sky-bank/auth_service/internal/repository"
	"github.com/mikhaeris/sky-bank/pkg/principal"
	v1 "github.com/mikhaeris/sky-bank/proto/gen/auth/v1"
	"go.opentelemetry.io/otel/trace"
)

type SessionsService struct {
	tracer trace.Tracer
	store  repository.DataStore
}

func NewSessionsService(
	tracer trace.Tracer,
	store repository.DataStore,
) *SessionsService {
	return &SessionsService{
		tracer: tracer,
		store:  store,
	}
}

func (s *SessionsService) GetSessions(ctx context.Context) ([]*v1.Session, error) {
	ctx, span := s.tracer.Start(ctx, "sessions.get")
	defer span.End()

	principal, err := principal.PrincipalFromContext(ctx)
	if err != nil {
		return nil, err
	}

	sessions, err := s.store.SessionRepository().GetSessionsByIdentityID(ctx, principal.IdentityID)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}

	return sessions, nil
}

func (s *SessionsService) RevokeSession(ctx context.Context, dto domain.RevokeSessionDTO) error {
	ctx, span := s.tracer.Start(ctx, "sessions.revoke")
	defer span.End()

	principal, err := principal.PrincipalFromContext(ctx)
	if err != nil {
		return err
	}

	err = s.store.SessionRepository().DeleteSessionByID(ctx, dto.SessionID, principal.IdentityID)
	if err != nil {
		if errors.Is(err, repository.ErrRecordNotFound) {
			return ErrSessionNotFound
		}
		return fmt.Errorf("revoke session: %w", err)
	}

	return nil
}

func (s *SessionsService) RevokeOtherSessions(ctx context.Context, dto domain.RevokeSessionDTO) error {
	ctx, span := s.tracer.Start(ctx, "sessions.revoke_other")
	defer span.End()

	principal, err := principal.PrincipalFromContext(ctx)
	if err != nil {
		return err
	}

	sessionRepo := s.store.SessionRepository()

	session, err := sessionRepo.GetById(ctx, dto.SessionID)
	if err != nil {
		return fmt.Errorf("revoke other sessions: %w", err)
	}

	if session.IdentityID != principal.IdentityID {
		return fmt.Errorf("revoke other session: wrong session")
	}

	err = sessionRepo.DeleteOtherSessionsByID(ctx, dto.SessionID, principal.IdentityID)
	if err != nil {
		return fmt.Errorf("revoke other sessions: %w", err)
	}

	return nil

}

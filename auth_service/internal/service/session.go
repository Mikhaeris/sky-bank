package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
	"github.com/mikhaeris/sky-bank/auth_service/internal/lib/principal"
	"github.com/mikhaeris/sky-bank/auth_service/internal/repository"
	v1 "github.com/mikhaeris/sky-bank/proto/gen/auth/v1"
)

type SessionsService struct {
	store repository.DataStore
}

func NewSessionsService(
	store repository.DataStore,
) *SessionsService {
	return &SessionsService{
		store: store,
	}
}

func (s *SessionsService) GetSessions(ctx context.Context) ([]*v1.Session, error) {
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
	principal, err := principal.PrincipalFromContext(ctx)
	if err != nil {
		return err
	}

	err = s.store.SessionRepository().DeleteOtherSessionsByID(ctx, dto.SessionID, principal.IdentityID)
	if err != nil {
		return fmt.Errorf("revoke other sessions: %w", err)
	}

	return nil

}

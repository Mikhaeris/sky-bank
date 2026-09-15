package service

import (
	"context"

	v1 "github.com/mikhaeris/sky-bank/auth_service/api/auth/v1"
	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
	"github.com/mikhaeris/sky-bank/auth_service/internal/lib/principal"
)

func (a *AuthService) GetSessions(ctx context.Context) ([]*v1.Session, error) {
	principal, err := principal.PrincipalFromContext(ctx)
	if err != nil {
		return nil, internalErr(err)
	}

	sessions, err := a.sessionRepo.GetSessionsByIdentityID(ctx, principal.IdentityID)
	if err != nil {
		return nil, internalErr(err)
	}

	return sessions, nil
}

func (a *AuthService) RevokeSession(ctx context.Context, dto domain.RevokeSessionDTO) error {
	principal, err := principal.PrincipalFromContext(ctx)
	if err != nil {
		return internalErr(err)
	}

	err = a.sessionRepo.DeleteSessionByID(ctx, dto.SessionID, principal.IdentityID)
	if err != nil {
		return internalErr(err)
	}

	return nil
}

func (a *AuthService) RevokeOtherSessions(ctx context.Context, dto domain.RevokeSessionDTO) error {
	principal, err := principal.PrincipalFromContext(ctx)
	if err != nil {
		return internalErr(err)
	}

	err = a.sessionRepo.DeleteOtherSessionsByID(ctx, dto.SessionID, principal.IdentityID)
	if err != nil {
		return internalErr(err)
	}

	return nil

}

package handler

import (
	"context"
	"uuid"

	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
	v1 "github.com/mikhaeris/sky-bank/proto/gen/auth/v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (a *AuthHandler) GetSessions(ctx context.Context, _ *emptypb.Empty) (*v1.GetSessionsResponse, error) {
	sessions, err := a.sessionsService.GetSessions(ctx)
	if err != nil {
		return nil, err
	}

	return &v1.GetSessionsResponse{
		Sessions: sessions,
	}, nil
}

func (a *AuthHandler) RevokeSession(ctx context.Context, in *v1.RevokeSessionRequest) (*emptypb.Empty, error) {
	id, err := uuid.Parse(in.SessionId)
	if err != nil {
		return nil, ErrInvalidID
	}

	dto := domain.RevokeSessionDTO{
		SessionID: id,
	}

	err = a.sessionsService.RevokeSession(ctx, dto)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (a *AuthHandler) RevokeOtherSessions(ctx context.Context, in *v1.RevokeSessionRequest) (*emptypb.Empty, error) {
	id, err := uuid.Parse(in.SessionId)
	if err != nil {
		return nil, ErrInvalidID
	}

	dto := domain.RevokeSessionDTO{
		SessionID: id,
	}

	err = a.sessionsService.RevokeOtherSessions(ctx, dto)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

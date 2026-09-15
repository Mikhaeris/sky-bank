package handler

import (
	"context"
	"fmt"
	"uuid"

	v1 "github.com/mikhaeris/sky-bank/auth_service/api/auth/v1"
	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (a *AuthHandler) GetSessions(ctx context.Context, _ *emptypb.Empty) (*v1.GetSessionsResponse, error) {
	sessions, err := a.authService.GetSessions(ctx)
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
		return &emptypb.Empty{}, fmt.Errorf("invalid id")
	}

	dto := domain.RevokeSessionDTO{
		SessionID: id,
	}

	err = a.authService.RevokeSession(ctx, dto)
	if err != nil {
		return &emptypb.Empty{}, err
	}

	return &emptypb.Empty{}, nil
}

func (a *AuthHandler) RevokeOtherSessions(ctx context.Context, in *v1.RevokeSessionRequest) (*emptypb.Empty, error) {
	id, err := uuid.Parse(in.SessionId)
	if err != nil {
		return &emptypb.Empty{}, fmt.Errorf("invalid id")
	}

	dto := domain.RevokeSessionDTO{
		SessionID: id,
	}

	err = a.authService.RevokeOtherSessions(ctx, dto)
	if err != nil {
		return &emptypb.Empty{}, err
	}

	return &emptypb.Empty{}, nil
}

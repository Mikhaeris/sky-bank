package handler

import (
	"context"
	"time"
	"uuid"

	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
	v1 "github.com/mikhaeris/sky-bank/proto/gen/auth/v1"
)

func (a *AuthHandler) StartAuthentication(ctx context.Context, in *v1.StartAuthenticationRequest) (*v1.StartAuthenticationResponse, error) {
	ctx, span := a.tracer.Start(ctx, "handler StartAuthentication")
	defer span.End()

	dto := domain.IdentityDTO{
		Email: in.Email,
	}

	challengeID, err := a.identityService.StartAuthentication(ctx, dto)
	if err != nil {
		return nil, err
	}

	return &v1.StartAuthenticationResponse{
		ChallengeId: challengeID.String(),
	}, nil
}

func (a *AuthHandler) CompleteAuthentication(ctx context.Context, in *v1.CompleteAuthenticationRequest) (*v1.CompleteAuthenticationResponse, error) {
	id, err := uuid.Parse(in.ChallengeId)
	if err != nil {
		return nil, ErrInvalidID
	}

	dto := domain.OtpDTO{
		ChallengeID:   id,
		CodePlaintext: in.OtpCode,
	}

	tokens, err := a.identityService.CompleteAuthentication(ctx, dto)
	if err != nil {
		return nil, err
	}

	return &v1.CompleteAuthenticationResponse{
		Tokens: &v1.Tokens{
			AccessToken:  tokens.Access,
			RefreshToken: tokens.Refresh,
			ExpiresAt:    tokens.ExpiresAt.Format(time.RFC3339Nano),
		},
	}, nil
}

func (a *AuthHandler) RefreshTokens(ctx context.Context, in *v1.RefreshTokensRequest) (*v1.RefreshTokensResponse, error) {
	dto := domain.TokensDTO{
		Refersh: in.RefreshToken,
	}

	tokens, err := a.identityService.RefreshTokens(ctx, dto)
	if err != nil {
		return nil, err
	}

	return &v1.RefreshTokensResponse{
		Tokens: &v1.Tokens{
			AccessToken:  tokens.Access,
			RefreshToken: tokens.Refresh,
			ExpiresAt:    tokens.ExpiresAt.Format(time.RFC3339Nano),
		},
	}, nil
}

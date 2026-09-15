package handler

import (
	"context"

	v1 "github.com/mikhaeris/sky-bank/auth_service/api/auth/v1"
	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (a *AuthHandler) StartAuthentication(ctx context.Context, in *v1.StartAuthenticationRequest) (*emptypb.Empty, error) {
	dto := domain.IdentityDTO{
		Email: in.Email,
	}

	err := a.authService.StartAuthentication(ctx, dto)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (a *AuthHandler) CompleteAuthentication(ctx context.Context, in *v1.CompleteAuthenticationRequest) (*v1.CompleteAuthenticationResponse, error) {
	dto := domain.OtpDto{
		Email:         in.Email,
		CodePlaintext: in.OtpCode,
	}

	tokens, err := a.authService.CompleteAuthentication(ctx, dto)
	if err != nil {
		return nil, err
	}

	return &v1.CompleteAuthenticationResponse{
		Tokens: &v1.Tokens{
			AccessToken:  tokens.Access,
			RefreshToken: tokens.Refresh,
		},
	}, nil
}

func (a *AuthHandler) RefreshTokens(ctx context.Context, in *v1.RefreshTokensRequest) (*v1.RefreshTokensResponse, error) {
	dto := domain.TokensDTO{
		Refersh: in.RefreshToken,
	}

	tokens, err := a.authService.RefreshTokens(ctx, dto)
	if err != nil {
		return nil, err
	}

	return &v1.RefreshTokensResponse{
		Tokens: &v1.Tokens{
			AccessToken:  tokens.Access,
			RefreshToken: tokens.Refresh,
		},
	}, nil
}

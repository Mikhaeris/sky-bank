package handler

import (
	"context"
	"fmt"
	"uuid"

	v1 "github.com/mikhaeris/sky-bank/auth_service/api/auth/v1"
	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
)

func (h *AuthHandler) CreateAuthenticationToken(ctx context.Context, in *v1.CreateAuthenticationTokenRequest) (*v1.CreateAuthenticationTokenResponse, error) {
	dto := domain.UserDTO{
		Email:    in.Email,
		Password: in.Password,
	}

	tokens, err := h.authService.CreateAuthenticationToken(ctx, dto)
	if err != nil {
		return nil, err
	}

	return &v1.CreateAuthenticationTokenResponse{
		Tokens: &v1.Tokens{
			AccessToken:  tokens.Access,
			RefreshToken: tokens.Refresh,
		},
	}, nil
}

func (h *AuthHandler) CreateActivationToken(ctx context.Context, in *v1.CreateActivationTokenRequest) (*v1.CreateActivationTokenResponse, error) {
	dto := domain.UserDTO{
		Email:    in.Email,
		Password: in.Password,
	}

	user, err := h.authService.CreateActivationToken(ctx, dto)
	if err != nil {
		return nil, err
	}

	return &v1.CreateActivationTokenResponse{
		UserUuid: user.Id.String(),
	}, nil
}

func (h *AuthHandler) CreatePasswordResetToken(ctx context.Context, in *v1.CreatePasswordResetTokenRequest) (*v1.CreatePasswordResetTokenResponse, error) {
	dto := domain.UserDTO{
		Email: in.Email,
	}

	user, err := h.authService.CreateResetPasswordToken(ctx, dto)
	if err != nil {
		return nil, err
	}

	return &v1.CreatePasswordResetTokenResponse{
		UserUuid: user.Id.String(),
	}, nil
}

func (h *AuthHandler) RefreshTokens(ctx context.Context, in *v1.RefreshTokensRequest) (*v1.RefreshTokensResponse, error) {
	dto := domain.TokensDTO{
		Access:  in.Tokens.AccessToken,
		Refresh: in.Tokens.RefreshToken,
	}

	tokens, err := h.authService.RefreshTokens(ctx, dto)
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

func (h *AuthHandler) LogOut(ctx context.Context, in *v1.LogOutRequest) (*v1.LogOutResponse, error) {
	userUuid, err := uuid.Parse(in.UserUuid)
	if err != nil {
		return nil, fmt.Errorf("bad user data")
	}

	dto := domain.UserLogOutDTO{
		Id: userUuid,
		TokensDTO: domain.TokensDTO{
			Access:  in.Tokens.AccessToken,
			Refresh: in.Tokens.RefreshToken,
		},
	}

	err = h.authService.LogOut(ctx, dto)
	if err != nil {
		return nil, err
	}

	return &v1.LogOutResponse{
		UserUuid: in.UserUuid,
	}, nil
}

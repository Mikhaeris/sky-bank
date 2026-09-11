package handler

import (
	"context"

	v1 "github.com/mikhaeris/sky-bank/auth_service/api/auth/v1"
)

func (h *AuthHandler) CreateAuthenticationToken(ctx context.Context, in *v1.CreateAuthenticationTokenRequest) (*v1.CreateAuthenticationTokenResponse, error) {
	// token, err := h.authService.CreateToken(r.Context(), user)
	// if err != nil {
	// 	h.serverErrorResponse(w, r, err)
	// 	return nil
	// }

	// return nil
	panic("not implemented")
}

func (h *AuthHandler) CreateActivationToken(ctx context.Context, in *v1.CreateActivationTokenRequest) (*v1.CreateActivationTokenResponse, error) {
	panic("not implemented")
}

func (h *AuthHandler) CreatePasswordResetToken(ctx context.Context, in *v1.CreatePasswordResetTokenRequest) (*v1.CreatePasswordResetTokenResponse, error) {
	panic("not implemented")
}

func (h *AuthHandler) RefreshTokens(ctx context.Context, in *v1.RefreshTokensRequest) (*v1.RefreshTokensResponse, error) {
	panic("not implemented")
}

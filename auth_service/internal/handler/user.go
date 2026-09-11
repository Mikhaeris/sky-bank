package handler

import (
	"context"

	authv1 "github.com/mikhaeris/sky-bank/auth_service/api/auth/v1"
	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
)

func (h *AuthHandler) RegisterUser(ctx context.Context, in *authv1.RegisterUserRequest) (*authv1.RegisterUserResponse, error) {
	h.logger.Info("get refister request")

	userDto := domain.UserDTO{
		Email:    in.Email,
		Password: in.Password,
	}

	userUUID, err := h.authService.RegisterUser(ctx, userDto)
	if err != nil {
		return &authv1.RegisterUserResponse{}, err
	}

	return &authv1.RegisterUserResponse{
		UserUuid: userUUID.String(),
	}, nil
}

func (h *AuthHandler) ActivateUser(ctx context.Context, in *authv1.ActivateUserRequest) (*authv1.ActivateUserResponse, error) {
	panic("not implemented")
}

func (h *AuthHandler) UpdateUserPassword(ctx context.Context, in *authv1.UpdateUserPasswordRequest) (*authv1.UpdateUserPasswordResponse, error) {
	panic("not implemented")
}

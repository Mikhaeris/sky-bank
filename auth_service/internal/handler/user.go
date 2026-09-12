package handler

import (
	"context"
	"fmt"

	authv1 "github.com/mikhaeris/sky-bank/auth_service/api/auth/v1"
	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
)

func (h *AuthHandler) RegisterUser(ctx context.Context, in *authv1.RegisterUserRequest) (*authv1.RegisterUserResponse, error) {
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
	dto := domain.ActivateUserDTO{
		TokenPlaintext: in.ActivationToken,
	}

	user, err := h.authService.ActivateUser(ctx, dto)
	if err != nil {
		return nil, fmt.Errorf(err.Error())
	}

	return &authv1.ActivateUserResponse{
		UserUuid: user.Id.String(),
	}, nil
}

func (h *AuthHandler) UpdateUserPassword(ctx context.Context, in *authv1.UpdateUserPasswordRequest) (*authv1.UpdateUserPasswordResponse, error) {
	dto := domain.ResetPasswordDTO{
		TokenPlaintext: in.PasswordResetToken,
		Password:       in.Password,
	}

	user, err := h.authService.ResetPassword(ctx, dto)
	if err != nil {
		return nil, err
	}

	return &authv1.UpdateUserPasswordResponse{
		UserUuid: user.Id.String(),
	}, nil
}

func (h *AuthHandler) RecoverCompromisedAccount(ctx context.Context, in *authv1.RecoverCompromisedAccountRequest) (*authv1.RecoverCompromisedAccountResponse, error) {
	dto := domain.ResetPasswordDTO{
		TokenPlaintext: in.SecurityToken,
		Password:       in.Password,
	}

	user, err := h.authService.RecoverCompromisedAccount(ctx, dto)
	if err != nil {
		return nil, err
	}

	return &authv1.RecoverCompromisedAccountResponse{
		UserUuid: user.Id.String(),
	}, nil
}

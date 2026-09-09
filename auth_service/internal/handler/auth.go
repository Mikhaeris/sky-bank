package handler

import (
	"context"
	"net/http"

	auth_service "github.com/mikhaeris/sky-bank/auth_service/api/auth/v1"
	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
)

func (h *AuthHandler) RegisterUser(ctx context.Context, in *auth_service.RegisterRequest) (*auth_service.RegisterResponse, error) {
	userDto := domain.UserDTO{
		Email:    in.Email,
		Password: in.Password,
	}

	userUUID, err := h.authService.RegisterUser(ctx, userDto)
	if err != nil {
		return &auth_service.RegisterResponse{}, err
	}

	return &auth_service.RegisterResponse{
		UserUUID: userUUID.String(),
	}, nil
}

func (h *AuthHandler) ActivateUser(w http.ResponseWriter, r *http.Request) {
	var activateUser domain.ActivateUserDTO

	err := h.ReadJSON(w, r, &activateUser)
	if err != nil {
		h.badRequestResponse(w, r, err)
		return
	}

	user, err := h.authService.ActivateUser(r.Context(), activateUser)
	if err != nil {
		h.serverErrorResponse(w, r, err)
		return
	}

	err = h.WriteJSON(w, http.StatusOK, Envelope{"user": user}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

func (h *AuthHandler) ResetPassword() {

}

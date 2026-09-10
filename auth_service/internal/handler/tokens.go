package handler

import (
	"net/http"

	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
)

func (h *AuthHandler) CreateAutentificationToken(w http.ResponseWriter, r *http.Request) {
	var user domain.UserDTO

	err := h.ReadJSON(w, r, &user)
	if err != nil {
		h.badRequestResponse(w, r, err)
		return
	}

	token, err := h.authService.CreateToken(r.Context(), user)
	if err != nil {
		h.serverErrorResponse(w, r, err)
		return
	}

	err = h.WriteJSON(w, http.StatusCreated, Envelope{"auth": token}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

func (h *AuthHandler) CreateActivationToken() {

}

func (h *AuthHandler) CreatePasswordResetToken() {

}

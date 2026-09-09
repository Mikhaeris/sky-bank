package handler

import (
	"net/http"

	"github.com/mikhaeris/bank-test/auth_service/internal/models"
)

func (h *AuthHandler) CreateAutentificationToken(w http.ResponseWriter, r *http.Request) {
	var user models.UserDTO

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

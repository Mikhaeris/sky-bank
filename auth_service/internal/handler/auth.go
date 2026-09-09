package handler

import (
	"net/http"

	"github.com/mikhaeris/bank-test/auth_service/internal/models"
)

func (h *AuthHandler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	var dto models.UserDTO

	err := h.ReadJSON(w, r, &dto)
	if err != nil {
		h.badRequestResponse(w, r, err)
		return
	}

	userUUID, err := h.authService.RegisterUser(r.Context(), dto)
	if err != nil {
		h.serverErrorResponse(w, r, err)
		return
	}

	// write answer
	err = h.WriteJSON(w, http.StatusCreated, Envelope{"user": userUUID}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

func (h *AuthHandler) ActivateUser(w http.ResponseWriter, r *http.Request) {
	var activateUser models.ActivateUserDTO

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

package handler

import (
	"net/http"
)

func (h *AuthHandler) logError(r *http.Request, err error) {
	var (
		method = r.Method
		uri    = r.URL.RequestURI()
	)

	h.logger.Error(err.Error(), "method", method, "uri", uri)
}

func (h *AuthHandler) errorResponses(w http.ResponseWriter, r *http.Request, status int, message any) {
	env := Envelope{"error": message}

	err := h.WriteJSON(w, status, env, nil)
	if err != nil {
		h.logError(r, err)
		w.WriteHeader(500)
	}
}

func (h *AuthHandler) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	h.logError(r, err)

	message := "the server encountered a problem and could not process your request"
	h.errorResponses(w, r, http.StatusInternalServerError, message)
}

func (h *AuthHandler) badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	h.errorResponses(w, r, http.StatusBadRequest, err.Error())
}

package main

import (
	"net/http"

	"github.com/mikhaeris/bank-test/auth_service/internal/handler"
)

func routes(handler *handler.AuthHandler) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/register", handler.RegisterUser)

	mux.HandleFunc("PUT /api/v1/tokens/authentication", handler.CreateAutentificationToken)

	// create refresh token
	// put refresh token here:
	// Set-Cookie: refresh_token=<token>; HttpOnly; Secure; SameSite=Strict; Path=/api/v1/tokens/refresh
	// put access token to json answer
	// POST /api/v1/tokens/refresh

	return mux
}

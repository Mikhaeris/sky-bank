package main

import (
	"context"
	"fmt"
	"net/http"

	auth_service "github.com/mikhaeris/sky-bank/auth_service/api/auth/v1"
)

type AuthHandler struct {
}

func (h *AuthHandler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	var userDTO UserDTO

	err := readJSON(w, r, &userDTO)
	if err != nil {
		return
	}

	out := auth_service.RegisterRequest{
		Email:    userDTO.Email,
		Password: userDTO.Password,
	}

	userUUID, err := GrpcClient.RegisterUser(context.Background(), &out)
	if err != nil {
		fmt.Fprint(w, "internal server error", "error", err)
		return
	}

	err = writeJSON(w, http.StatusCreated, envelope{"userUUID": userUUID}, nil)
	if err != nil {
		fmt.Fprint(w, "server error")
	}
}

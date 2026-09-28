package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	authv1 "github.com/mikhaeris/sky-bank/proto/gen/auth/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const (
	cookieName = "__Secure-refresh"
	cookiePath = "/api/v1/auth/refresh"
)

type AuthHandler struct {
	Auth authv1.AuthClient
	Mux  *runtime.ServeMux
}

func (h *AuthHandler) Complete(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")

	var req authv1.CompleteAuthenticationRequest
	if !decode(w, r, &req) {
		return
	}

	resp, err := h.Auth.CompleteAuthentication(r.Context(), &req)
	if err != nil {
		h.rpcError(w, r, err)
		return
	}

	h.writeTokens(w, r, resp.GetTokens())
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")

	cookie, err := r.Cookie(cookieName)
	if err != nil || cookie.Value == "" {
		http.Error(w, "refresh token required", http.StatusUnauthorized)
		return
	}

	resp, err := h.Auth.RefreshTokens(r.Context(), &authv1.RefreshTokensRequest{
		RefreshToken: cookie.Value,
	})
	if err != nil {
		switch status.Code(err) {
		case codes.Unauthenticated:
			clearRefreshCookie(w)
		case codes.InvalidArgument:
			clearRefreshCookie(w)
			err = status.Error(codes.Unauthenticated, "invalid refresh token")
		}
		h.rpcError(w, r, err)
		return
	}

	h.writeTokens(w, r, resp.GetTokens())
}

func (h *AuthHandler) writeTokens(w http.ResponseWriter, r *http.Request, tokens *authv1.Tokens) {
	if tokens == nil || tokens.AccessToken == "" || tokens.RefreshToken == "" {
		h.rpcError(w, r, status.Error(codes.Internal, "empty token response"))
		return
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, tokens.GetExpiresAt())
	if err != nil || !expiresAt.After(time.Now()) {
		h.rpcError(w, r, status.Error(codes.Internal, "invalid refresh token expiry"))
		return
	}
	maxAge := int((time.Until(expiresAt) + time.Second - 1) / time.Second)

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    tokens.RefreshToken,
		Path:     cookiePath,
		Expires:  expiresAt,
		MaxAge:   maxAge,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Tokens struct {
			AccessToken string `json:"accessToken"`
		} `json:"tokens"`
	}{
		Tokens: struct {
			AccessToken string `json:"accessToken"`
		}{AccessToken: tokens.AccessToken},
	})
}

func clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Path:     cookiePath,
		MaxAge:   -1,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

func decode(w http.ResponseWriter, r *http.Request, dst proto.Message) bool {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 4096))
	if err == nil {
		err = protojson.Unmarshal(body, dst)
	}
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return false
	}
	return true
}

func (h *AuthHandler) rpcError(w http.ResponseWriter, r *http.Request, err error) {
	_, marshaler := runtime.MarshalerForRequest(h.Mux, r)
	runtime.HTTPError(r.Context(), h.Mux, marshaler, w, r, err)
}

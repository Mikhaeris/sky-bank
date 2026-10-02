package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	authv1 "github.com/mikhaeris/sky-bank/proto/gen/auth/v1"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type refreshErrorClient struct {
	authv1.AuthClient
	err error
}

func (c refreshErrorClient) RefreshTokens(context.Context, *authv1.RefreshTokensRequest, ...grpc.CallOption) (*authv1.RefreshTokensResponse, error) {
	return nil, c.err
}

func TestRefreshPreservesAuthErrorDetails(t *testing.T) {
	backendStatus, err := status.New(codes.Unauthenticated, "invalid refresh token").WithDetails(
		&errdetails.ErrorInfo{Reason: "INVALID_REFRESH_TOKEN", Domain: "auth_service"},
	)
	if err != nil {
		t.Fatal(err)
	}

	h := AuthHandler{Auth: refreshErrorClient{err: backendStatus.Err()}, Mux: runtime.NewServeMux()}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: cookieName, Value: "expired-token"})
	w := httptest.NewRecorder()

	h.Refresh(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("HTTP status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
	var body struct {
		Code    int32  `json:"code"`
		Message string `json:"message"`
		Details []struct {
			Reason string `json:"reason"`
			Domain string `json:"domain"`
		} `json:"details"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Code != int32(codes.Unauthenticated) || body.Message != "invalid refresh token" {
		t.Fatalf("response = %+v", body)
	}
	if len(body.Details) != 1 || body.Details[0].Reason != "INVALID_REFRESH_TOKEN" || body.Details[0].Domain != "auth_service" {
		t.Fatalf("error details = %+v", body.Details)
	}

	cookies := w.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != cookieName || cookies[0].MaxAge >= 0 {
		t.Fatalf("refresh cookie was not cleared: %+v", cookies)
	}
}

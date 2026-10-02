package integration_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
	"github.com/mikhaeris/sky-bank/auth_service/internal/lib/jwt"
	"github.com/mikhaeris/sky-bank/auth_service/internal/service"
	ke "github.com/mikhaeris/sky-bank/pkg/kafkaevents"
	"go.opentelemetry.io/otel/trace/noop"
)

type loginNotifications struct{}

func (loginNotifications) Produce(context.Context, string, ke.BaseEvent) error { return nil }

type authenticationFixture struct {
	*otpTestEnv
	auth *service.IdentityService
}

type authenticationState struct {
	challenges int
	outbox     int
	identities int
	sessions   int
}

func newAuthenticationFixture(t *testing.T) *authenticationFixture {
	t.Helper()
	env := newOTPTestEnv(t)
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	keyPath := filepath.Join(t.TempDir(), "signing.pem")
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0600); err != nil {
		t.Fatal(err)
	}
	keys, err := jwt.NewKeys(keyPath, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	f := &authenticationFixture{otpTestEnv: env}
	f.auth = f.newService(keys)
	return f
}

func (f *authenticationFixture) newService(keys *jwt.Keys) *service.IdentityService {
	return service.NewIdentityService(
		noop.NewTracerProvider().Tracer("test"), keys, loginNotifications{}, f.service, f.store,
	)
}

func (f *authenticationFixture) issue(t *testing.T, email string) domain.OtpDTO {
	t.Helper()
	id, err := f.auth.StartAuthentication(f.ctx, domain.IdentityDTO{Email: email})
	if err != nil {
		t.Fatal(err)
	}
	var outbox domain.OTPOutbox
	if err := f.pool.QueryRow(f.ctx,
		`SELECT event_id, encrypted_event FROM otp_outbox WHERE challenge_id = $1`, id,
	).Scan(&outbox.EventID, &outbox.EncryptedEvent); err != nil {
		t.Fatal(err)
	}
	event, err := f.cipher.Decrypt(outbox.EncryptedEvent, outbox.EventID, id)
	if err != nil {
		t.Fatal(err)
	}
	code, ok := event.Payload.Data["otpCode"].(string)
	if !ok || code == "" {
		t.Fatalf("OTP event has no code: %+v", event.Payload.Data)
	}
	return domain.OtpDTO{ChallengeID: id, CodePlaintext: code}
}

func (f *authenticationFixture) assertState(t *testing.T, email string, dto domain.OtpDTO, want authenticationState) {
	t.Helper()
	var got [4]int
	err := f.pool.QueryRow(f.ctx, `SELECT
		(SELECT count(*) FROM challenges WHERE id = $1),
		(SELECT count(*) FROM otp_outbox WHERE challenge_id = $1),
		(SELECT count(*) FROM identities WHERE email = $2),
		(SELECT count(*) FROM sessions s JOIN identities i ON i.id = s.identity_id WHERE i.email = $2)`,
		dto.ChallengeID, email,
	).Scan(&got[0], &got[1], &got[2], &got[3])
	if err != nil {
		t.Fatal(err)
	}
	if expected := [4]int{want.challenges, want.outbox, want.identities, want.sessions}; got != expected {
		t.Fatalf("challenge/outbox/identity/session counts = %v, want %v", got, expected)
	}
}

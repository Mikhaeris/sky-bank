package unit_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/mikhaeris/sky-bank/auth_service/internal/config"
)

func TestOtpLimitsReadFromYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	data := []byte(`otp_limits:
  initial_issues: 5
  initial_cooldown: 15s
  max_cooldown: 5m
  issue_reset_after: 30m
  max_attempts_per_challenge: 5
`)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}

	var cfg config.Config
	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		t.Fatal(err)
	}
	limits := cfg.OtpLimits
	if limits.InitialIssues != 5 ||
		limits.InitialCooldown != 15*time.Second ||
		limits.MaxCooldown != 5*time.Minute ||
		limits.IssueResetAfter != 30*time.Minute ||
		limits.MaxAttemptsPerChallenge != 5 {
		t.Fatalf("unexpected OTP limits: %+v", limits)
	}
}

func TestOtpIssueDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("otp_limits: {}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var cfg config.Config
	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		t.Fatal(err)
	}
	l := cfg.OtpLimits
	if l.InitialIssues != 5 || l.InitialCooldown != 15*time.Second || l.MaxCooldown != 5*time.Minute || l.IssueResetAfter != 30*time.Minute || l.MaxAttemptsPerChallenge != 5 {
		t.Fatalf("unexpected issue defaults: %+v", l)
	}
}

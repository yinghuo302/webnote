package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitReadsSystemdCredentialAndDefaultsSecureCookie(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "webnote.env")
	if err := os.WriteFile(path, []byte("WEBNOTE_DATABASE_URL=postgres:///webnote?host=/var/run/postgresql\nWEBNOTE_SMTP_HOST=smtp.example.test\nWEBNOTE_SMTP_USER=mailer@example.test\nWEBNOTE_SMTP_PASSWORD=test-secret\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CREDENTIALS_DIRECTORY", dir)
	for _, key := range []string{"WEBNOTE_DATABASE_URL", "WEBNOTE_SMTP_HOST", "WEBNOTE_SMTP_USER", "WEBNOTE_SMTP_PASSWORD", "WEBNOTE_SECURE_COOKIES", "WEBNOTE_PORT", "WEBNOTE_SMTP_PORT"} {
		t.Setenv(key, "")
	}
	if err := Init(); err != nil {
		t.Fatal(err)
	}
	if !Conf.SecureCookies {
		t.Fatal("secure cookies must default on")
	}
	if Conf.DatabaseURL == "" || Conf.Email.Auth != "test-secret" {
		t.Fatal("systemd credential was not read")
	}
	if strings.Contains(os.Getenv("WEBNOTE_SMTP_PASSWORD"), "test-secret") {
		t.Fatal("test secret unexpectedly in environment")
	}
}

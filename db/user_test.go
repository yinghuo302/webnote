package db

import (
	"net/url"
	"os"
	"strings"
	"testing"
	"webnote/config"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func openIsolatedPostgres(t *testing.T) {
	t.Helper()
	databaseURL := os.Getenv("WEBNOTE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set WEBNOTE_TEST_DATABASE_URL")
	}
	admin, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	adminDB, err := admin.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = adminDB.Close() })
	schema := "webnote_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if err := admin.Exec(`CREATE SCHEMA "` + schema + `"`).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = admin.Exec(`DROP SCHEMA "` + schema + `" CASCADE`).Error })
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	config.Conf = &config.Config{DatabaseURL: parsed.String()}
	if err := InitDB(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = Close() })
}

func TestCodeRetryLimitAndCooldown(t *testing.T) {
	openIsolatedPostgres(t)
	email := "tester@example.test"
	if err := SetCode(email, "123456"); err != nil {
		t.Fatal(err)
	}
	if err := SetCode(email, "111111"); err == nil {
		t.Fatal("code resend cooldown missing")
	}
	for i := 0; i < 5; i++ {
		if _, err := CreateUser(email, "a-long-test-password", "000000"); err == nil {
			t.Fatal("invalid code accepted")
		}
	}
	if _, err := CreateUser(email, "a-long-test-password", "123456"); err == nil {
		t.Fatal("correct code accepted after five failed tries")
	}
	var count int64
	if err := dbCon.Model(&User{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatal("attempted registrations created a user", err)
	}
}

func TestCreateUserAcceptsIssuedVerificationCode(t *testing.T) {
	openIsolatedPostgres(t)
	const email = "Registration-Test@example.test"
	const code = "482061"
	const password = "a-long-test-password"
	if err := SetCode(email, code); err != nil {
		t.Fatal(err)
	}
	id, err := CreateUser(email, password, "  "+code+"\n")
	if err != nil {
		t.Fatalf("correct issued code was rejected: %v", err)
	}
	if id == 0 {
		t.Fatal("registration returned zero user id")
	}
	if _, err := UserLogin(email, password); err != nil {
		t.Fatalf("newly registered user cannot sign in: %v", err)
	}
	var codeCount int64
	if err := dbCon.Model(&Code{}).Count(&codeCount).Error; err != nil || codeCount != 0 {
		t.Fatalf("verification code was not consumed: count=%d err=%v", codeCount, err)
	}
}

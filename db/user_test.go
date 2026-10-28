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

func TestCodeRetryLimitAndCooldown(t *testing.T) {
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
	defer adminDB.Close()
	schema := "webnote_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if err := admin.Exec(`CREATE SCHEMA "` + schema + `"`).Error; err != nil {
		t.Fatal(err)
	}
	defer admin.Exec(`DROP SCHEMA "` + schema + `" CASCADE`)
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
	defer Close()
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

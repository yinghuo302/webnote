package main

import (
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"webnote/config"
	"webnote/db"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestPreflightDetectsConflictsWithoutWriting(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("migration requires Python 3")
	}
	source := filepath.Join(t.TempDir(), "old.db")
	dataDir := t.TempDir()
	script := `import sqlite3,sys
c=sqlite3.connect(sys.argv[1]);c.executescript("CREATE TABLE users (user_id INTEGER); CREATE TABLE logins (email TEXT); CREATE TABLE files (file_id TEXT, content_stored INTEGER); INSERT INTO users VALUES (1); INSERT INTO logins VALUES ('A@example.org'), ('a@example.org'); INSERT INTO files VALUES ('aaaaaaa1-aaaa-aaaa-aaaa-aaaaaaaaaaaa',0);");c.close()`
	if output, err := exec.Command("python3", "-c", script, source).CombinedOutput(); err != nil {
		t.Fatalf("create fixture: %v: %s", err, output)
	}
	stats, err := preflight(source, dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Counts["users"] != 1 || stats.Counts["logins"] != 2 || stats.Counts["files"] != 1 || stats.DuplicateEmailGroups != 1 || stats.MissingMarkdown != 1 {
		t.Fatalf("unexpected preflight: %+v", stats)
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatal("source database damaged", err)
	}
}
func TestImportRowsIntoIsolatedPostgreSQLSchema(t *testing.T) {
	dsn := os.Getenv("WEBNOTE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set WEBNOTE_TEST_DATABASE_URL to run PostgreSQL import tests")
	}
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	adminSQL, err := admin.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer adminSQL.Close()
	schema := "webnote_import_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if err := admin.Exec(`CREATE SCHEMA "` + schema + `"`).Error; err != nil {
		t.Fatal(err)
	}
	defer admin.Exec(`DROP SCHEMA "` + schema + `" CASCADE`)
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	values := parsed.Query()
	values.Set("search_path", schema)
	parsed.RawQuery = values.Encode()
	config.Conf = &config.Config{DatabaseURL: parsed.String()}
	if err := db.InitDB(); err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	dataDir := t.TempDir()
	id := uuid.New()
	if err := os.WriteFile(filepath.Join(dataDir, id.String()+".md"), []byte("# Legacy"), 0600); err != nil {
		t.Fatal(err)
	}
	stream := `{"table":"users","row":{"user_id":7,"nickname":"Old"}}` + "\n" + `{"table":"logins","row":{"email":"old@example.org","passwd":"oldpass","user_id":7}}` + "\n" + `{"table":"files","row":{"user_id":7,"file_id":"` + id.String() + `","name":"Old note","public":1}}` + "\n"
	if err := db.GetDB().Transaction(func(tx *gorm.DB) error { return importRows(tx, strings.NewReader(stream), dataDir) }); err != nil {
		t.Fatal(err)
	}
	var note db.Files
	if err := db.GetDB().First(&note, "file_id = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	if note.Content != "# Legacy" || !note.ContentStored || !note.Public || note.UserId != 7 {
		t.Fatalf("incorrect note import: %+v", note)
	}
	var user db.User
	if err := db.GetDB().Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if user.UserId <= 7 {
		t.Fatal("PostgreSQL user id sequence was not advanced")
	}
}

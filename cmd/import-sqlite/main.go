// import-sqlite performs a one-time, offline migration of a legacy Webnote SQLite database.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"webnote/config"
	"webnote/db"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const exporter = `import json, sqlite3, sys, urllib.parse, os
source = os.path.abspath(sys.argv[1])
uri = 'file:' + urllib.parse.quote(source, safe='/') + '?mode=ro'
con = sqlite3.connect(uri, uri=True)
con.row_factory = sqlite3.Row
for table in ('users', 'logins', 'files'):
    for row in con.execute('SELECT * FROM ' + table):
        print(json.dumps({'table': table, 'row': dict(row)}, ensure_ascii=False), flush=True)
con.close()
`

const preflightScript = `import json, sqlite3, sys, urllib.parse, os
source = os.path.abspath(sys.argv[1])
data_dir = os.path.abspath(sys.argv[2])
con = sqlite3.connect('file:' + urllib.parse.quote(source, safe='/') + '?mode=ro', uri=True)
counts = {table: con.execute('SELECT count(*) FROM ' + table).fetchone()[0] for table in ('users', 'logins', 'files')}
duplicates = con.execute('SELECT count(*) FROM (SELECT lower(trim(email)) FROM logins GROUP BY lower(trim(email)) HAVING count(*) > 1)').fetchone()[0]
missing = sum(not os.path.isfile(os.path.join(data_dir, row[0] + '.md')) for row in con.execute('SELECT file_id FROM files WHERE file_id IS NOT NULL AND (content_stored = 0 OR content_stored IS NULL)' if 'content_stored' in (column[1] for column in con.execute('PRAGMA table_info(files)')) else 'SELECT file_id FROM files WHERE file_id IS NOT NULL'))
print(json.dumps({'counts': counts, 'duplicate_email_groups': duplicates, 'missing_markdown': missing}))
con.close()
`

type preflightResult struct {
	Counts               map[string]int `json:"counts"`
	DuplicateEmailGroups int            `json:"duplicate_email_groups"`
	MissingMarkdown      int            `json:"missing_markdown"`
}

func preflight(source, dataDir string) (preflightResult, error) {
	output, err := exec.Command("python3", "-c", preflightScript, source, dataDir).Output()
	if err != nil {
		return preflightResult{}, fmt.Errorf("inspect SQLite database: %w", err)
	}
	var result preflightResult
	err = json.Unmarshal(output, &result)
	return result, err
}

type record struct {
	Table string          `json:"table"`
	Row   json.RawMessage `json:"row"`
}
type legacyUser struct {
	ID          int64  `json:"user_id"`
	Nickname    string `json:"nickname"`
	Description string `json:"description"`
	Avatar      string `json:"avatar"`
}
type legacyLogin struct {
	Email    string `json:"email"`
	Password string `json:"passwd"`
	UserID   int64  `json:"user_id"`
}
type legacyFile struct {
	UserID      int64     `json:"user_id"`
	Name        string    `json:"name"`
	FileID      uuid.UUID `json:"file_id"`
	Content     *string   `json:"content"`
	Stored      int       `json:"content_stored"`
	Description string    `json:"description"`
	Public      int       `json:"public"`
}

func importRows(tx *gorm.DB, reader io.Reader, dataDir string) error {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 32*1024*1024)
	counts := map[string]int{}
	for scanner.Scan() {
		var rec record
		if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
			return err
		}
		switch rec.Table {
		case "users":
			var old legacyUser
			if err := json.Unmarshal(rec.Row, &old); err != nil {
				return err
			}
			if err := tx.Create(&db.User{UserId: old.ID, Nickname: old.Nickname, Description: old.Description, Avatar: old.Avatar}).Error; err != nil {
				return err
			}
		case "logins":
			var old legacyLogin
			if err := json.Unmarshal(rec.Row, &old); err != nil {
				return err
			}
			if err := tx.Create(&db.Login{Email: old.Email, Passwd: old.Password, UserId: old.UserID}).Error; err != nil {
				return err
			}
		case "files":
			var old legacyFile
			if err := json.Unmarshal(rec.Row, &old); err != nil {
				return err
			}
			if old.FileID == uuid.Nil {
				return errors.New("legacy note missing file_id")
			}
			text := ""
			if old.Content != nil {
				text = *old.Content
			}
			if old.Stored == 0 {
				data, err := os.ReadFile(filepath.Join(dataDir, old.FileID.String()+".md"))
				if err != nil {
					return fmt.Errorf("legacy note %s: %w", old.FileID, err)
				}
				text = string(data)
			}
			note := db.Files{UserId: old.UserID, Name: old.Name, FileId: old.FileID, Content: text, ContentStored: true, Description: old.Description, Public: old.Public != 0}
			if err := tx.Create(&note).Error; err != nil {
				return err
			}
		default:
			return fmt.Errorf("unexpected source table %q", rec.Table)
		}
		counts[rec.Table]++
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if err := tx.Exec(`SELECT setval(pg_get_serial_sequence('users','user_id'), GREATEST(COALESCE((SELECT MAX(user_id) FROM users),1),1), (SELECT COUNT(*)>0 FROM users))`).Error; err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "imported %d users, %d logins and %d notes\n", counts["users"], counts["logins"], counts["files"])
	return nil
}
func run() error {
	source := flag.String("sqlite", "", "offline SQLite source database path")
	dataDir := flag.String("data-dir", "./data", "old Markdown data directory")
	check := flag.Bool("check", false, "read-only inspection; do not connect to PostgreSQL or import data")
	flag.Parse()
	if *source == "" {
		return errors.New("--sqlite is required")
	}
	if _, err := os.Stat(*source); err != nil {
		return err
	}
	stats, err := preflight(*source, *dataDir)
	if err != nil {
		return err
	}
	fmt.Printf("SQLite preflight: %d users, %d logins, %d notes; %d duplicate email groups; %d missing Markdown files\n", stats.Counts["users"], stats.Counts["logins"], stats.Counts["files"], stats.DuplicateEmailGroups, stats.MissingMarkdown)
	if *check {
		return nil
	}
	if stats.DuplicateEmailGroups > 0 {
		return errors.New("duplicate emails require a data-owner decision; no data was imported")
	}
	if stats.MissingMarkdown > 0 {
		return errors.New("missing legacy Markdown files; no data was imported")
	}
	dsn := os.Getenv("WEBNOTE_DATABASE_URL")
	if dsn == "" {
		return errors.New("WEBNOTE_DATABASE_URL is required")
	}
	config.Conf = &config.Config{DatabaseURL: dsn}
	if err := db.InitDB(); err != nil {
		return err
	}
	defer db.Close()
	cmd := exec.Command("python3", "-c", exporter, *source)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	err = db.GetDB().Transaction(func(tx *gorm.DB) error {
		for _, model := range []interface{}{&db.User{}, &db.Login{}, &db.Files{}} {
			var count int64
			if err := tx.Model(model).Count(&count).Error; err != nil {
				return err
			}
			if count != 0 {
				return errors.New("target database is not empty; import into a new database")
			}
		}
		if err := importRows(tx, stdout, *dataDir); err != nil {
			return err
		}
		if err := cmd.Wait(); err != nil {
			return fmt.Errorf("SQLite export: %w: %s", err, stderr.String())
		}
		return nil
	})
	if err != nil {
		cmd.Process.Kill()
		_ = cmd.Wait()
		return err
	}
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "import failed:", err)
		os.Exit(1)
	}
}

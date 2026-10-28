package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"webnote/config"
	"webnote/db"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestNotesAuthorizationAndPersistence(t *testing.T) {
	databaseURL := os.Getenv("WEBNOTE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set WEBNOTE_TEST_DATABASE_URL to a disposable PostgreSQL database to run integration tests")
	}
	admin, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	adminSQL, err := admin.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer adminSQL.Close()
	schema := "webnote_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if err := admin.Exec(`CREATE SCHEMA "` + schema + `"`).Error; err != nil {
		t.Fatal(err)
	}
	defer admin.Exec(`DROP SCHEMA "` + schema + `" CASCADE`)
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	params := parsed.Query()
	params.Set("search_path", schema)
	parsed.RawQuery = params.Encode()
	config.Conf = &config.Config{DataDir: t.TempDir(), ImgDir: t.TempDir(), DatabaseURL: parsed.String()}
	if err := db.InitDB(); err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.GetDB().Create(&db.User{Nickname: "Alice"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.GetDB().Create(&db.Login{Email: "alice@example.com", Passwd: "legacy-pass", UserId: 1}).Error; err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	Init(engine)
	request := func(method, path, body, cookie string) *httptest.ResponseRecorder {
		t.Helper()
		var input *bytes.Reader
		if body == "" {
			input = bytes.NewReader(nil)
		} else {
			input = bytes.NewReader([]byte(body))
		}
		req := httptest.NewRequest(method, path, input)
		if method != "GET" && method != "HEAD" {
			req.Header.Set("Origin", "http://example.com")
		}
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		if cookie != "" {
			req.Header.Set("Cookie", cookie)
		}
		res := httptest.NewRecorder()
		engine.ServeHTTP(res, req)
		return res
	}
	if res := request("GET", "/", "", ""); res.Code != 404 {
		t.Fatalf("backend unexpectedly serves frontend: %d", res.Code)
	}
	if res := request("GET", "/api/private/files", "", ""); res.Code != 401 {
		t.Fatalf("anonymous list: %d", res.Code)
	}
	login := request("POST", "/api/public/signin", `{"email":"alice@example.com","passwd":"legacy-pass"}`, "")
	if login.Code != 200 {
		t.Fatalf("login: %d %s", login.Code, login.Body)
	}
	cookie := login.Header().Get("Set-Cookie")
	if cookie == "" {
		t.Fatal("missing session cookie")
	}
	if strings.Contains(cookie, "alice@example.com") || strings.Contains(cookie, "userId") || !strings.Contains(cookie, "HttpOnly") || !strings.Contains(cookie, "SameSite=Strict") {
		t.Fatalf("unsafe session cookie attributes: %q", cookie)
	}
	if len(login.Result().Cookies()) != 1 || len(login.Result().Cookies()[0].Value) != 64 {
		t.Fatal("expected 256-bit opaque session token")
	}
	var info db.Login
	if err := db.GetDB().Where("email = ?", "alice@example.com").First(&info).Error; err != nil || !strings.HasPrefix(info.Passwd, "$2") {
		t.Fatal("legacy password was not upgraded to bcrypt", err)
	}
	saved := request("POST", "/api/private/file", `{"name":"First","description":"test","content":"# Hello"}`, cookie)
	if saved.Code != 200 {
		t.Fatalf("save: %d %s", saved.Code, saved.Body)
	}
	var result struct {
		FileID string `json:"fileId"`
	}
	if err := json.Unmarshal(saved.Body.Bytes(), &result); err != nil || result.FileID == "" {
		t.Fatal(saved.Body.String(), err)
	}
	if err := db.GetDB().Create(&db.User{Nickname: "Bob"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.GetDB().Create(&db.Login{Email: "bob@example.com", Passwd: "another-legacy", UserId: 2}).Error; err != nil {
		t.Fatal(err)
	}
	other := request("POST", "/api/public/signin", `{"email":"bob@example.com","passwd":"another-legacy"}`, "")
	if other.Code != 200 {
		t.Fatalf("second login: %d", other.Code)
	}
	otherCookie := other.Header().Get("Set-Cookie")
	if res := request("GET", "/api/private/file?id="+result.FileID, "", otherCookie); res.Code != 404 {
		t.Fatalf("cross user read: %d", res.Code)
	}
	if res := request("POST", "/api/private/file", `{"fileId":"`+result.FileID+`","name":"stolen","content":"stolen"}`, otherCookie); res.Code != 404 {
		t.Fatalf("cross user write: %d", res.Code)
	}
	article := request("GET", "/api/public/article?id="+result.FileID, "", "")
	if article.Code != 404 {
		t.Fatalf("private note leaked: %d", article.Code)
	}
	read := request("GET", "/api/private/file?id="+result.FileID, "", cookie)
	if read.Code != 200 || !strings.Contains(read.Body.String(), "# Hello") {
		t.Fatalf("read: %d %s", read.Code, read.Body)
	}
	if res := request("POST", "/api/private/share?id="+result.FileID, "", cookie); res.Code != 200 {
		t.Fatalf("share: %d", res.Code)
	}
	article = request("GET", "/api/public/article?id="+result.FileID, "", "")
	if article.Code != 200 || !strings.Contains(article.Body.String(), "# Hello") {
		t.Fatalf("shared note: %d %s", article.Code, article.Body)
	}
	if res := request("POST", "/api/private/file", `{"fileId":"`+result.FileID+`","name":"First","content":""}`, cookie); res.Code != 200 {
		t.Fatalf("empty overwrite: %d %s", res.Code, res.Body)
	}
	read = request("GET", "/api/private/file?id="+result.FileID, "", cookie)
	if read.Code != 200 || !strings.Contains(read.Body.String(), `"content":""`) {
		t.Fatalf("empty overwrite read: %s", read.Body)
	}
	bad := httptest.NewRequest(http.MethodPost, "/api/private/file", bytes.NewBufferString(`{"name":"bad"}`))
	bad.Header.Set("Origin", "http://attacker.example")
	bad.Header.Set("Cookie", cookie)
	bad.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	engine.ServeHTTP(res, bad)
	if res.Code != 403 {
		t.Fatalf("cross origin: %d", res.Code)
	}
	missingOrigin := httptest.NewRequest(http.MethodPost, "/api/public/signin", bytes.NewBufferString(`{"email":"x","passwd":"x"}`))
	missingOrigin.Header.Set("Content-Type", "application/json")
	res = httptest.NewRecorder()
	engine.ServeHTTP(res, missingOrigin)
	if res.Code != 403 {
		t.Fatalf("missing origin accepted: %d", res.Code)
	}
	if res := request("GET", "/api/public/img/../../notes.db", "", ""); res.Code == 200 {
		t.Fatal("path traversal exposed database")
	}
	if res := request("POST", "/api/private/logout", "", cookie); res.Code != 200 {
		t.Fatalf("logout: %d", res.Code)
	}
	if res := request("GET", "/api/private/user", "", cookie); res.Code != 401 {
		t.Fatalf("stolen cookie survived logout: %d", res.Code)
	}
	login = request("POST", "/api/public/signin", `{"email":"alice@example.com","passwd":"legacy-pass"}`, "")
	if login.Code != 200 {
		t.Fatalf("relogin: %d", login.Code)
	}
	cookie = login.Header().Get("Set-Cookie")
	if res := request("POST", "/api/private/user", `{"oldPasswd":"legacy-pass","passwd":"a-longer-secret-password"}`, cookie); res.Code != 200 {
		t.Fatalf("password change: %d %s", res.Code, res.Body)
	}
	if res := request("GET", "/api/private/user", "", cookie); res.Code != 401 {
		t.Fatalf("cookie survived password change: %d", res.Code)
	}
	login = request("POST", "/api/public/signin", `{"email":"alice@example.com","passwd":"a-longer-secret-password"}`, "")
	if login.Code != 200 {
		t.Fatalf("new password login: %d", login.Code)
	}
	first := login.Header().Get("Set-Cookie")
	second := request("POST", "/api/public/signin", `{"email":"alice@example.com","passwd":"a-longer-secret-password"}`, "")
	if second.Code != 200 {
		t.Fatalf("second device login: %d", second.Code)
	}
	otherSession := second.Header().Get("Set-Cookie")
	if res := request("POST", "/api/private/logout-all", "", first); res.Code != 200 {
		t.Fatalf("all-device logout: %d", res.Code)
	}
	for _, old := range []string{first, otherSession} {
		if res := request("GET", "/api/private/user", "", old); res.Code != 401 {
			t.Fatalf("device cookie survived global logout: %d", res.Code)
		}
	}
}

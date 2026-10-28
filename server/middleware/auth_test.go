package middleware

import (
	"net/http/httptest"
	"strings"
	"testing"
	"webnote/config"

	"github.com/gin-gonic/gin"
)

func TestSecureCookieAttributes(t *testing.T) {
	config.Conf = &config.Config{SecureCookies: true}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	SetSessionCookie(c, strings.Repeat("a", 64))
	cookie := c.Writer.Header().Get("Set-Cookie")
	for _, part := range []string{"__Host-webnote_session=", "Secure", "HttpOnly", "SameSite=Strict", "Path=/", "Max-Age=86400"} {
		if !strings.Contains(cookie, part) {
			t.Fatalf("missing %q in secure cookie", part)
		}
	}
	if strings.Contains(cookie, "Domain=") {
		t.Fatal("__Host- cookie must not have Domain")
	}
}

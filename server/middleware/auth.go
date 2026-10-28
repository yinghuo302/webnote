package middleware

import (
	"net/http"
	"time"
	"webnote/config"
	"webnote/db"

	"github.com/gin-gonic/gin"
)

const defaultCookieName = "webnote_session"
const secureCookieName = "__Host-webnote_session"

func CookieName() string {
	if config.Conf.SecureCookies {
		return secureCookieName
	}
	return defaultCookieName
}
func SetSessionCookie(c *gin.Context, token string) {
	http.SetCookie(c.Writer, &http.Cookie{Name: CookieName(), Value: token, Path: "/", MaxAge: int(db.SessionTTL.Seconds()), Expires: time.Now().Add(db.SessionTTL), HttpOnly: true, Secure: config.Conf.SecureCookies, SameSite: http.SameSiteStrictMode})
	c.Header("Cache-Control", "no-store")
}
func ClearSessionCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{Name: CookieName(), Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: config.Conf.SecureCookies, SameSite: http.SameSiteStrictMode})
	c.Header("Cache-Control", "no-store")
}
func AuthRequired(c *gin.Context) {
	token, err := c.Cookie(CookieName())
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"status": "unauthorized"})
		return
	}
	uid, err := db.LookupSession(token)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"status": "unauthorized"})
		return
	}
	c.Set("userId", uid)
	c.Set("sessionToken", token)
	c.Next()
}

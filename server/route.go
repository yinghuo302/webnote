package server

import (
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"webnote/config"
	"webnote/server/handlers"
	"webnote/server/middleware"

	"github.com/gin-gonic/gin"
)

var attempts = struct {
	sync.Mutex
	items map[string][]time.Time
}{items: make(map[string][]time.Time)}

func throttle(limit int) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP() + ":" + c.FullPath()
		now := time.Now()
		attempts.Lock()
		if len(attempts.items) > 10000 {
			attempts.items = make(map[string][]time.Time)
		}
		entries := attempts.items[key]
		fresh := entries[:0]
		for _, t := range entries {
			if now.Sub(t) < time.Minute {
				fresh = append(fresh, t)
			}
		}
		if len(fresh) >= limit {
			attempts.items[key] = fresh
			attempts.Unlock()
			c.AbortWithStatusJSON(429, gin.H{"status": "请求过于频繁"})
			return
		}
		attempts.items[key] = append(fresh, now)
		attempts.Unlock()
		c.Next()
	}
}
func sameOrigin(c *gin.Context) {
	if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead || c.Request.Method == http.MethodOptions {
		return
	}
	origin := c.GetHeader("Origin")
	if origin == "" {
		c.AbortWithStatusJSON(403, gin.H{"status": "origin required"})
		return
	}
	if origin != "" {
		u, err := url.Parse(origin)
		if err != nil || !strings.EqualFold(u.Host, c.Request.Host) || (config.Conf.SecureCookies && u.Scheme != "https") {
			c.AbortWithStatusJSON(403, gin.H{"status": "origin forbidden"})
			return
		}
	}
	if c.GetHeader("Sec-Fetch-Site") == "cross-site" {
		c.AbortWithStatusJSON(403, gin.H{"status": "cross-site forbidden"})
		return
	}
}
func Init(e *gin.Engine) {
	e.Use(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Cache-Control", "no-store")
		c.Next()
	}, sameOrigin)
	g := e.Group("/api")
	public := g.Group("/public")
	private := g.Group("/private")
	private.Use(middleware.AuthRequired)
	g.GET("/ping", func(c *gin.Context) { c.String(200, "pong") })
	public.POST("/signin", throttle(10), handlers.Login)
	public.POST("/signup", throttle(5), handlers.Register)
	public.POST("/code", throttle(3), handlers.GetCode)
	private.POST("/logout", handlers.Logout)
	private.POST("/logout-all", handlers.LogoutAll)
	public.GET("/img/:name", handlers.GetImg)
	private.POST("/img", handlers.UploadImg)
	private.POST("/avatar", handlers.UploadAvatar)
	private.POST("/share", handlers.ShareFile)
	private.GET("/user", handlers.GetUser)
	private.POST("/user", handlers.UpdataUserInfo)
	private.GET("/files", handlers.GetFiles)
	private.GET("/file", handlers.GetFile)
	private.POST("/file", handlers.SaveFile)
	private.DELETE("/file", handlers.DeleteFile)
	public.GET("/articles", handlers.GetArticles)
	public.GET("/article", handlers.GetArticle)
}

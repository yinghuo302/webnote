package server

import (
	"webnote/server/handlers"
	"webnote/server/middleware"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func Init(e *gin.Engine) {
	store := cookie.NewStore([]byte("secret"))
	e.Use(sessions.Sessions("LoginSession", store))
	g := e.Group("/api")
	public := g.Group("/public")
	private := g.Group("/private")
	g.Any("/ping", func(c *gin.Context) {
		c.String(200, "pong")
	})
	private.Use(middleware.AuthRequired)
	public.POST("/signin", handlers.Login)
	public.POST("/signup", handlers.Register)
	public.GET("/code", handlers.GetCode)
	private.GET("/logout", handlers.Logout)
	public.GET("/img/*path", handlers.GetImg)
	public.POST("/img", handlers.UploadImg)
	private.POST("/avatar", handlers.UploadAvatar)
	private.POST("/share", handlers.ShareFile)
	private.GET("/user", handlers.GetUser)
	private.POST("/user", handlers.UpdataUserInfo)
	private.GET("/files", handlers.GetFiles)
	private.GET("/file", handlers.GetFile)
	private.POST("/file", handlers.SaveFile)
	private.GET("/share", handlers.ShareFile)
	public.GET("/articles", handlers.GetArticles)
	public.GET("/article", handlers.GetArticle)
}

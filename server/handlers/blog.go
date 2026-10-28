package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
	"strconv"
	"webnote/db"
)

func GetArticles(c *gin.Context) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 || page > 100000 {
		fail(c, 400, "页码无效")
		return
	}
	keyword := c.Query("keywd")
	if len(keyword) > 200 {
		fail(c, 400, "搜索词过长")
		return
	}
	count, err := db.GetArticleCount(keyword)
	if err != nil {
		fail(c, 500, "查询失败")
		return
	}
	articles, err := db.GetArticles(keyword, page)
	if err != nil {
		fail(c, 500, "查询失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "pages": (count + 9) / 10, "articles": articles})
}
func GetArticle(c *gin.Context) {
	id, err := uuid.Parse(c.Query("id"))
	if err != nil {
		fail(c, 400, "文章ID无效")
		return
	}
	article, err := db.GetArticle(id)
	if err != nil {
		fail(c, 404, "文章未公开或不存在")
		return
	}
	article.User, err = db.GetUser(article.UserId)
	if err != nil {
		fail(c, 500, "读取用户失败")
		return
	}
	article.Content, err = legacyContent(article)
	if err != nil {
		fail(c, 500, "读取文章失败")
		return
	}
	c.JSON(200, gin.H{"status": "success", "article": article})
}

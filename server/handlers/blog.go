package handlers

import (
	"net/http"
	"os"
	"strconv"
	"webnote/db"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func GetArticles(ctx *gin.Context) {
	pageNum, err := strconv.Atoi(ctx.Query("page"))
	keywd := ctx.Query("keywd")
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"status": "参数获取失败"})
		return
	}
	cnt := int64(0)
	if pageNum == 1 {
		cnt, err = db.GetArticleCount(keywd)
		if err != nil {
			ctx.JSON(http.StatusOK, gin.H{"status": "文章数量获取失败"})
			return
		}
	}
	artiles, err := db.GetArticles(keywd, pageNum)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"status": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"pages":    cnt / 10,
		"articles": artiles,
	})
}

func GetArticle(ctx *gin.Context) {
	fileId, err := uuid.Parse(ctx.Query("id"))
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"status": "文章ID不正确"})
		return
	}
	article, err := db.GetArticle(fileId)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"status": err.Error()})
		return
	}
	if !article.Public {
		ctx.JSON(http.StatusOK, gin.H{"status": "文章未公开"})
		return
	}
	article.User, err = db.GetUser(article.UserId)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"status": err.Error()})
		return
	}
	content, err := os.ReadFile("./data/" + article.FileId.String() + ".md")
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"status": "文件打开失败"})
		return
	}
	article.Content = string(content)
	ctx.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"article": article,
	})
}

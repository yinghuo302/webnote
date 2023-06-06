package handlers

import (
	"net/http"
	"os"
	"path"
	"webnote/config"
	"webnote/db"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func GetFiles(ctx *gin.Context) {
	userId := ctx.GetInt64("userId")
	files, err := db.GetFiles(userId)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"status": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"status": "success",
		"files":  files,
	})
}

func GetFile(ctx *gin.Context) {
	userId := ctx.GetInt64("userId")
	fileId, err := uuid.Parse(ctx.Query("id"))
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"status": "文件ID错误"})
		return
	}
	file, err := db.CheckFileUser(fileId, userId)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"status": err.Error()})
		return
	}
	content, err := os.ReadFile(config.Conf.DataDir + "/" + file.FileId.String() + ".md")
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"status": "文件打开失败"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"content": string(content),
	})
}

func SaveFile(ctx *gin.Context) {
	userId := ctx.GetInt64("userId")
	file := &db.Files{}
	ctx.ShouldBind(file)
	if file.FileId != uuid.Nil {
		_, err := db.CheckFileUser(file.FileId, userId)
		if err != nil {
			ctx.JSON(http.StatusOK, gin.H{"status": err.Error()})
			return
		}
	}
	file.UserId = userId
	uuidFile, err := db.SaveFile(file)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"status": err.Error()})
		return
	}
	err = os.WriteFile(config.Conf.DataDir+"/"+file.FileId.String()+".md", []byte(file.Content), 0600)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"status": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"status": "success",
		"fileId": uuidFile.String(),
	})
}

func DeleteFile(ctx *gin.Context) {
	userId := ctx.GetInt64("userId")
	fileId, err := uuid.Parse(ctx.Query("id"))
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"status": "文件ID错误"})
		return
	}
	_, err = db.CheckFileUser(fileId, userId)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"status": err.Error()})
		return
	}
	err = db.DeleteFile(fileId)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"status": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"status": "success"})
}

func ShareFile(ctx *gin.Context) {
	userId := ctx.GetInt64("userId")
	fileId, err := uuid.Parse(ctx.Query("id"))
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"status": "文件ID错误"})
		return
	}
	_, err = db.CheckFileUser(fileId, userId)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"status": err.Error()})
		return
	}
	err = db.ShareFile(fileId)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"status": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"status": "success"})
}

func UploadImg(ctx *gin.Context) {
	header, err := ctx.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"status": "文件上传失败"})
		return
	}
	filename := uuid.New().String() + path.Ext(header.Filename)
	err = ctx.SaveUploadedFile(header, config.Conf.ImgDir+"/"+filename)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"status": "文件保存失败"})
		return
	}
	ctx.JSON(200, gin.H{
		"status": "success",
		"url":    "/api/img/" + filename,
	})
}

func UploadAvatar(ctx *gin.Context) {
	header, err := ctx.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"status": "文件上传失败"})
		return
	}
	filename := uuid.New().String() + path.Ext(header.Filename)
	err = ctx.SaveUploadedFile(header, config.Conf.ImgDir+"/"+filename)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"status": "文件保存失败"})
		return
	}
	user := &db.User{UserId: ctx.GetInt64("userId"), Avatar: "/api/public/img/" + filename}
	db.UpdateUser(user)
	ctx.JSON(200, gin.H{
		"status": "success",
		"url":    "/api/public/img/" + filename,
	})
}

func GetImg(ctx *gin.Context) {
	file, exist := ctx.Params.Get("path")
	if !exist {
		ctx.JSON(http.StatusBadRequest, gin.H{"status": "请输入文件名"})
		return
	}
	ctx.File(config.Conf.ImgDir + file)
}

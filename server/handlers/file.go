package handlers

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"webnote/config"
	"webnote/db"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func fail(c *gin.Context, status int, message string) { c.JSON(status, gin.H{"status": message}) }
func legacyContent(file *db.Files) (string, error) {
	if file.ContentStored {
		return file.Content, nil
	}
	bytes, err := os.ReadFile(filepath.Join(config.Conf.DataDir, file.FileId.String()+".md"))
	return string(bytes), err
}
func GetFiles(c *gin.Context) {
	files, err := db.GetFiles(c.GetInt64("userId"))
	if err != nil {
		fail(c, 500, "获取笔记失败")
		return
	}
	c.JSON(200, gin.H{"status": "success", "files": files})
}
func GetFile(c *gin.Context) {
	id, err := uuid.Parse(c.Query("id"))
	if err != nil {
		fail(c, 400, "文件ID错误")
		return
	}
	file, err := db.CheckFileUser(id, c.GetInt64("userId"))
	if err != nil {
		fail(c, 404, "文件不存在")
		return
	}
	text, err := legacyContent(file)
	if err != nil {
		fail(c, 500, "读取笔记失败")
		return
	}
	c.JSON(200, gin.H{"status": "success", "content": text})
}
func SaveFile(c *gin.Context) {
	var file db.Files
	if c.ShouldBindJSON(&file) != nil {
		fail(c, 400, "无效请求")
		return
	}
	if len(file.Name) == 0 || len(file.Name) > 200 || len(file.Description) > 4000 || len(file.Content) > 1024*1024 {
		fail(c, 400, "笔记内容或标题不合法（正文最大 1 MiB）")
		return
	}
	file.UserId = c.GetInt64("userId")
	file.Public = false
	if file.FileId != uuid.Nil {
		if _, err := db.CheckFileUser(file.FileId, file.UserId); err != nil {
			fail(c, 404, "文件不存在")
			return
		}
	}
	id, err := db.SaveFile(&file)
	if err != nil {
		fail(c, 500, "保存笔记失败")
		return
	}
	c.JSON(200, gin.H{"status": "success", "fileId": id.String()})
}
func DeleteFile(c *gin.Context) {
	id, err := uuid.Parse(c.Query("id"))
	if err != nil {
		fail(c, 400, "文件ID错误")
		return
	}
	if _, err = db.CheckFileUser(id, c.GetInt64("userId")); err != nil {
		fail(c, 404, "文件不存在")
		return
	}
	if db.DeleteFile(id) != nil {
		fail(c, 500, "删除失败")
		return
	}
	c.JSON(200, gin.H{"status": "success"})
}
func ShareFile(c *gin.Context) {
	id, err := uuid.Parse(c.Query("id"))
	if err != nil {
		fail(c, 400, "文件ID错误")
		return
	}
	if _, err = db.CheckFileUser(id, c.GetInt64("userId")); err != nil {
		fail(c, 404, "文件不存在")
		return
	}
	if db.ShareFile(id) != nil {
		fail(c, 500, "分享失败")
		return
	}
	c.JSON(200, gin.H{"status": "success"})
}
func saveImage(c *gin.Context) (string, error) {
	header, err := c.FormFile("file")
	if err != nil {
		return "", err
	}
	if header.Size > 5<<20 || header.Size <= 0 {
		return "", http.ErrContentLength
	}
	src, err := header.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()
	buf := make([]byte, 512)
	n, err := src.Read(buf)
	if err != nil && err != io.EOF {
		return "", err
	}
	mime := http.DetectContentType(buf[:n])
	extensions := map[string]string{"image/png": ".png", "image/jpeg": ".jpg", "image/gif": ".gif", "image/webp": ".webp"}
	ext, ok := extensions[mime]
	if !ok {
		return "", http.ErrNotSupported
	}
	if err := os.MkdirAll(config.Conf.ImgDir, 0700); err != nil {
		return "", err
	}
	name := uuid.NewString() + ext
	dest, err := os.OpenFile(filepath.Join(config.Conf.ImgDir, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", err
	}
	defer dest.Close()
	if _, err = dest.Write(buf[:n]); err == nil {
		_, err = io.Copy(dest, io.LimitReader(src, 5<<20))
	}
	if err != nil {
		os.Remove(dest.Name())
		return "", err
	}
	return "/api/public/img/" + name, nil
}
func UploadImg(c *gin.Context) {
	url, err := saveImage(c)
	if err != nil {
		fail(c, 400, "仅允许 5 MiB 以内的 PNG、JPEG、GIF、WebP 图片")
		return
	}
	c.JSON(200, gin.H{"status": "success", "url": url})
}
func UploadAvatar(c *gin.Context) {
	url, err := saveImage(c)
	if err != nil {
		fail(c, 400, "图片无效")
		return
	}
	if db.UpdateUser(&db.User{UserId: c.GetInt64("userId"), Avatar: url}) != nil {
		fail(c, 500, "更新头像失败")
		return
	}
	c.JSON(200, gin.H{"status": "success", "url": url})
}
func GetImg(c *gin.Context) {
	name := c.Param("name")
	ext := filepath.Ext(name)
	if len(name) != 36+len(ext) || (ext != ".png" && ext != ".jpg" && ext != ".jpeg" && ext != ".gif" && ext != ".webp") {
		fail(c, 404, "图片不存在")
		return
	}
	if _, err := uuid.Parse(name[:36]); err != nil {
		fail(c, 404, "图片不存在")
		return
	}
	c.Header("X-Content-Type-Options", "nosniff")
	c.File(filepath.Join(config.Conf.ImgDir, name))
}

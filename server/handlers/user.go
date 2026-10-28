package handlers

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net/mail"
	"strings"
	"webnote/config"
	"webnote/db"
	"webnote/utils"

	"github.com/gin-gonic/gin"
	"webnote/server/middleware"
)

type User struct {
	Email       string `json:"email"`
	Passwd      string `json:"passwd"`
	OldPasswd   string `json:"oldPasswd"`
	Code        string `json:"code"`
	Description string `json:"description"`
	Nickname    string `json:"nickname"`
}

func validEmail(email string) bool {
	address, err := mail.ParseAddress(email)
	return err == nil && strings.EqualFold(address.Address, email) && len(email) <= 254
}
func Login(ctx *gin.Context) {
	var user User
	if ctx.ShouldBindJSON(&user) != nil {
		ctx.JSON(400, gin.H{"status": "无效请求"})
		return
	}
	uid, err := db.UserLogin(user.Email, user.Passwd)
	if err != nil {
		ctx.JSON(401, gin.H{"status": "邮箱或密码错误"})
		return
	}
	if old, err := ctx.Cookie(middleware.CookieName()); err == nil {
		_ = db.DeleteSession(old)
	}
	token, err := db.CreateSession(uid)
	if err != nil {
		ctx.JSON(500, gin.H{"status": "会话保存失败"})
		return
	}
	middleware.SetSessionCookie(ctx, token)
	ctx.JSON(200, gin.H{"status": "success"})
}
func Register(ctx *gin.Context) {
	var user User
	if ctx.ShouldBindJSON(&user) != nil || !validEmail(user.Email) {
		ctx.JSON(400, gin.H{"status": "无效邮箱或请求"})
		return
	}
	uid, err := db.CreateUser(user.Email, user.Passwd, user.Code)
	if err != nil {
		message := "验证码错误或已过期"
		if !errors.Is(err, db.ErrEmailRegistered) && !strings.Contains(err.Error(), "验证码") && !strings.Contains(err.Error(), "密码") {
			message = "注册失败"
		}
		ctx.JSON(400, gin.H{"status": message})
		return
	}
	if old, err := ctx.Cookie(middleware.CookieName()); err == nil {
		_ = db.DeleteSession(old)
	}
	token, err := db.CreateSession(uid)
	if err != nil {
		ctx.JSON(500, gin.H{"status": "会话保存失败"})
		return
	}
	middleware.SetSessionCookie(ctx, token)
	ctx.JSON(200, gin.H{"status": "success"})
}
func GetCode(ctx *gin.Context) {
	email := strings.ToLower(strings.TrimSpace(ctx.Query("email")))
	if !validEmail(email) {
		ctx.JSON(400, gin.H{"status": "邮箱不正确"})
		return
	}
	if config.Conf.Email.Host == "" || config.Conf.Email.User == "" || config.Conf.Email.Auth == "" {
		ctx.JSON(503, gin.H{"status": "邮件服务未配置"})
		return
	}
	number, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		ctx.JSON(500, gin.H{"status": "验证码生成失败"})
		return
	}
	code := fmt.Sprintf("%06d", number.Int64())
	if err := db.SetCode(email, code); err != nil {
		if errors.Is(err, db.ErrEmailRegistered) {
			ctx.JSON(200, gin.H{"status": "success"})
			return
		}
		if errors.Is(err, db.ErrCodeCooldown) {
			ctx.JSON(429, gin.H{"status": "请求过于频繁"})
			return
		}
		ctx.JSON(500, gin.H{"status": "验证码保存失败"})
		return
	}
	if err := utils.SendMail(email, "Webnote 注册验证码", "验证码："+code+"，10 分钟内有效"); err != nil {
		ctx.JSON(502, gin.H{"status": "验证码发送失败"})
		return
	}
	ctx.JSON(200, gin.H{"status": "success"})
}
func Logout(ctx *gin.Context) {
	if err := db.DeleteSession(ctx.GetString("sessionToken")); err != nil {
		ctx.JSON(500, gin.H{"status": "退出失败"})
		return
	}
	middleware.ClearSessionCookie(ctx)
	ctx.JSON(200, gin.H{"status": "success"})
}
func LogoutAll(ctx *gin.Context) {
	if err := db.DeleteUserSessions(ctx.GetInt64("userId")); err != nil {
		ctx.JSON(500, gin.H{"status": "会话撤销失败"})
		return
	}
	middleware.ClearSessionCookie(ctx)
	ctx.JSON(200, gin.H{"status": "success"})
}
func GetUser(ctx *gin.Context) {
	user, err := db.GetUser(ctx.GetInt64("userId"))
	if err != nil {
		ctx.JSON(404, gin.H{"status": "用户不存在"})
		return
	}
	ctx.JSON(200, gin.H{"status": "success", "user": user})
}
func UpdataUserInfo(ctx *gin.Context) {
	var user User
	if ctx.ShouldBindJSON(&user) != nil {
		ctx.JSON(400, gin.H{"status": "无效请求"})
		return
	}
	if user.Email != "" {
		ctx.JSON(400, gin.H{"status": "暂不支持修改邮箱"})
		return
	}
	id := ctx.GetInt64("userId")
	if user.Passwd != "" {
		if err := db.UpdatePassword(id, user.OldPasswd, user.Passwd); err != nil {
			ctx.JSON(400, gin.H{"status": err.Error()})
			return
		}
		if err := db.DeleteUserSessions(id); err != nil {
			ctx.JSON(500, gin.H{"status": "密码已更改，但会话清理失败"})
			return
		}
		middleware.ClearSessionCookie(ctx)
	}
	if len(user.Nickname) > 100 || len(user.Description) > 2000 {
		ctx.JSON(400, gin.H{"status": "个人资料过长"})
		return
	}
	if user.Nickname != "" || user.Description != "" {
		if err := db.UpdateUser(&db.User{UserId: id, Nickname: user.Nickname, Description: user.Description}); err != nil {
			ctx.JSON(500, gin.H{"status": "更新失败"})
			return
		}
	}
	ctx.JSON(200, gin.H{"status": "success"})
}

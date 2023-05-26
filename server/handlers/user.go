package handlers

import (
	"math/rand"
	"net/http"
	"strconv"
	"webnote/db"

	"webnote/utils"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type User struct {
	Email       string `json:"email"`
	Passwd      string `json:"passwd"`
	Code        string `json:"code"`
	Description string `json:"description"`
	Nickname    string `json:"nickname"`
}

func Login(ctx *gin.Context) {
	session := sessions.Default(ctx)
	user := &User{}
	ctx.ShouldBind(user)
	uid, err := db.UserLogin(user.Email, user.Passwd)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"status": err.Error()})
		return
	}
	session.Set("userId", uid)
	session.Set("email", user.Email)
	if err := session.Save(); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"status": "success"})
}

func Register(ctx *gin.Context) {
	session := sessions.Default(ctx)
	user := &User{}
	ctx.ShouldBind(user)
	userId, err := db.CreateUser(user.Email, user.Passwd, user.Code)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"status": err.Error()})
		return
	}
	session.Set("userId", userId)
	session.Set("email", user.Email)
	if err := session.Save(); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"status": "Failed to save session"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"status": "success",
	})
}

func GetCode(ctx *gin.Context) {
	email := ctx.Query("email")
	code := strconv.Itoa(int(rand.Int31n(900000) + 100000))

	err := utils.SendMail(email, "Webnote注册重置认证", "本次的验证码为"+code+",有限期为10分钟")
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"status": "验证码发送失败"})
		return
	}
	err = db.SetCode(email, code)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"status": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"status": "success"})
}

func Logout(ctx *gin.Context) {
	session := sessions.Default(ctx)
	session.Delete("userId")
	session.Delete("email")
	if err := session.Save(); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"status": "Failed to save session"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"status": "success"})
}

func GetUser(ctx *gin.Context) {
	userId := ctx.GetInt64("userId")
	user, err := db.GetUser(userId)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"status": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"status": "success",
		"user":   user,
	})
}

func UpdataUserInfo(ctx *gin.Context) {
	user := &User{}
	ctx.ShouldBind(user)
	if len(user.Nickname) != 0 || len(user.Description) != 0 {
		userId := ctx.GetInt64("userId")
		err := db.UpdateUser(&db.User{Nickname: user.Nickname, Description: user.Description, UserId: userId})
		if err != nil {
			ctx.JSON(http.StatusOK, gin.H{"status": err.Error()})
			return
		}
	}
	if len(user.Passwd) != 0 || len(user.Email) != 0 {
		email := ctx.GetString("email")
		err := db.UpdateLogin(email, &db.Login{Email: user.Email, Passwd: user.Passwd})
		if err != nil {
			ctx.JSON(http.StatusOK, gin.H{"status": err.Error()})
			return
		}
	}
	ctx.JSON(http.StatusOK, gin.H{"status": "success"})
}

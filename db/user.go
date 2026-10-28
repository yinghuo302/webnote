package db

import (
	"crypto/subtle"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrInvalidCredentials = errors.New("邮箱或密码错误")
var ErrCodeCooldown = errors.New("请等待 60 秒后再次请求验证码")
var ErrEmailRegistered = errors.New("邮箱已注册")

func normalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }
func UserLogin(email, passwd string) (int64, error) {
	var info Login
	if err := dbCon.Where("LOWER(email) = ?", normalizeEmail(email)).First(&info).Error; err != nil {
		return 0, ErrInvalidCredentials
	}
	if strings.HasPrefix(info.Passwd, "$2") {
		if bcrypt.CompareHashAndPassword([]byte(info.Passwd), []byte(passwd)) != nil {
			return 0, ErrInvalidCredentials
		}
	} else {
		if subtle.ConstantTimeCompare([]byte(info.Passwd), []byte(passwd)) != 1 {
			return 0, ErrInvalidCredentials
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(passwd), bcrypt.DefaultCost)
		if err != nil {
			return 0, err
		}
		if err := dbCon.Model(&Login{}).Where("email = ? AND passwd = ?", info.Email, info.Passwd).Update("passwd", string(hash)).Error; err != nil {
			return 0, err
		}
	}
	return info.UserId, nil
}
func CreateUser(email, passwd, code string) (int64, error) {
	email = normalizeEmail(email)
	if len(passwd) < 12 || len(passwd) > 72 {
		return 0, errors.New("密码长度应为 12 至 72 个字符")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(passwd), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	attempt := dbCon.Model(&Code{}).Where("email = ? AND updated_at > ? AND attempts < 5", email, time.Now().Add(-10*time.Minute)).UpdateColumn("attempts", gorm.Expr("attempts + 1"))
	if attempt.Error != nil {
		return 0, attempt.Error
	}
	if attempt.RowsAffected == 0 {
		return 0, errors.New("验证码错误或已过期")
	}
	var uid int64
	err = dbCon.Transaction(func(tx *gorm.DB) error {
		var record Code
		if err := tx.Where("email = ?", email).First(&record).Error; err != nil {
			return errors.New("验证码错误或已过期")
		}
		if record.UpdatedAt.Add(10*time.Minute).Before(time.Now()) || bcrypt.CompareHashAndPassword([]byte(record.Code), []byte(code)) != nil {
			return errors.New("验证码错误或已过期")
		}
		var existing int64
		if err := tx.Model(&Login{}).Where("LOWER(email) = ?", email).Count(&existing).Error; err != nil {
			return err
		}
		if existing != 0 {
			return errors.New("邮箱已注册")
		}
		user := User{Nickname: "默认用户名", Description: "", Avatar: ""}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		if err := tx.Create(&Login{Email: email, Passwd: string(hash), UserId: user.UserId}).Error; err != nil {
			return errors.New("邮箱已注册")
		}
		if err := tx.Delete(&record).Error; err != nil {
			return err
		}
		uid = user.UserId
		return nil
	})
	return uid, err
}
func SetCode(email, code string) error {
	email = normalizeEmail(email)
	hash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return dbCon.Transaction(func(tx *gorm.DB) error {
		var registered int64
		if err := tx.Model(&Login{}).Where("LOWER(email) = ?", email).Count(&registered).Error; err != nil {
			return err
		}
		if registered > 0 {
			return ErrEmailRegistered
		}
		var old Code
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("email = ?", email).First(&old).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err == nil && time.Since(old.UpdatedAt) < time.Minute {
			return ErrCodeCooldown
		}
		return tx.Save(&Code{Email: email, Code: string(hash), Attempts: 0, UpdatedAt: time.Now()}).Error
	})
}
func GetUser(id int64) (*User, error) {
	user := &User{}
	err := dbCon.First(user, id).Error
	return user, err
}
func UpdateUser(user *User) error {
	return dbCon.Model(&User{}).Where("user_id = ?", user.UserId).Updates(user).Error
}
func UpdatePassword(id int64, oldPass, newPass string) error {
	if len(newPass) < 12 || len(newPass) > 72 {
		return errors.New("密码长度应为 12 至 72 个字符")
	}
	var info Login
	if err := dbCon.Where("user_id = ?", id).First(&info).Error; err != nil {
		return ErrInvalidCredentials
	}
	if strings.HasPrefix(info.Passwd, "$2") {
		if bcrypt.CompareHashAndPassword([]byte(info.Passwd), []byte(oldPass)) != nil {
			return ErrInvalidCredentials
		}
	} else if subtle.ConstantTimeCompare([]byte(info.Passwd), []byte(oldPass)) != 1 {
		return ErrInvalidCredentials
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPass), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	result := dbCon.Model(&Login{}).Where("user_id = ? AND passwd = ?", id, info.Passwd).Update("passwd", string(hash))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrInvalidCredentials
	}
	return nil
}

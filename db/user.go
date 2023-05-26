package db

import (
	"time"

	"github.com/pkg/errors"
)

func UserLogin(email string, passwd string) (int64, error) {
	loginInfo := &Login{}
	if err := dbCon.Where(&Login{Email: email}).First(loginInfo).Error; err != nil {
		return 0, errors.Wrapf(err, "Check Code")
	}
	if loginInfo.Passwd != passwd {
		return 0, errors.Errorf("密码错误")
	}
	return loginInfo.UserId, nil
}

func CreateUser(email string, passwd string, code string) (int64, error) {
	codeInfo := &Code{}
	if err := dbCon.Where(&Code{Email: email}).First(codeInfo).Error; err != nil {
		return 0, errors.Errorf("Check Code")
	}
	diff, _ := time.ParseDuration("10m")
	if codeInfo.Code != code {
		return 0, errors.Errorf("验证码错误")
	}
	if codeInfo.UpdatedAt.Add(diff).Before(time.Now()) {
		return 0, errors.Errorf("验证码过期")
	}
	user := &User{Nickname: "默认用户名", Description: "这个人很懒，没有描述", Avatar: "/api/public/img/default.png"}
	if err := dbCon.Create(user).Error; err != nil {
		return 0, errors.Errorf("用户创建失败")
	}
	login := &Login{Email: email, Passwd: passwd, UserId: user.UserId}
	if err := dbCon.Create(login).Error; err != nil {
		return 0, errors.Errorf("用户创建失败")
	}
	dbCon.Delete(codeInfo)
	return user.UserId, nil
}

func SetCode(email string, code string) error {
	res := dbCon.Where(&Code{Email: email}).Updates(&Code{Code: code})
	if res.Error == nil && res.RowsAffected != 0 {
		return nil
	}
	if err := dbCon.Create(&Code{Email: email, Code: code}).Error; err != nil {
		return err
	}
	return nil
}

func GetUser(userId int64) (*User, error) {
	user := &User{}
	if err := dbCon.First(user, userId).Error; err != nil {
		return nil, errors.New("用户获取失败")
	}
	return user, nil
}

func UpdateUser(user *User) error {
	if err := dbCon.Model(user).Updates(user).Error; err != nil {
		return errors.New("用户信息修改失败")
	}
	return nil
}

func UpdateLogin(email string, login *Login) error {
	if err := dbCon.Model(&Login{}).Where("email = ?", email).Updates(login).Error; err != nil {
		return errors.New("用户信息修改失败")
	}
	return nil
}

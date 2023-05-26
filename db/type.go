package db

import (
	"time"

	"github.com/google/uuid"
)

type Login struct {
	Email  string `json:"email" gorm:"index"`
	Passwd string `json:"passwd"`
	UserId int64  `json:"userId"`
}

type Code struct {
	Email     string    `json:"email" gorm:"primarykey"`
	Code      string    `json:"code"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

type User struct {
	UserId      int64  `json:"userId" gorm:"autoIncrement;primarykey"`
	Nickname    string `json:"nickname"`
	Description string `json:"description"`
	Avatar      string `json:"avatar"`
}

type Files struct {
	UserId      int64     `json:"userId" gorm:"index"`
	Name        string    `json:"name"`
	FileId      uuid.UUID `json:"fileId" gorm:"primarykey"`
	Content     string    `json:"content" gorm:"-" sql:"-"`
	Description string    `json:"description" gorm:"column:description"`
	User        *User     `json:"user" gorm:"-" sql:"-"`
	Public      bool      `json:"-" gorm:"column:public"`
}

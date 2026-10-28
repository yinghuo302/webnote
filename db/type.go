package db

import (
	"time"

	"github.com/google/uuid"
)

type Login struct {
	Email  string `json:"email" gorm:"uniqueIndex"`
	Passwd string `json:"-"`
	UserId int64  `json:"userId"`
}

type Code struct {
	Email     string    `json:"email" gorm:"primarykey"`
	Code      string    `json:"-"`
	Attempts  int       `json:"-" gorm:"default:0"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

type User struct {
	UserId      int64  `json:"userId" gorm:"autoIncrement;primarykey"`
	Nickname    string `json:"nickname"`
	Description string `json:"description"`
	Avatar      string `json:"avatar"`
}

type Files struct {
	UserId        int64     `json:"userId" gorm:"index"`
	Name          string    `json:"name"`
	FileId        uuid.UUID `json:"fileId" gorm:"type:uuid;primaryKey"`
	Content       string    `json:"content" gorm:"type:text"`
	ContentStored bool      `json:"-" gorm:"column:content_stored"`
	Description   string    `json:"description" gorm:"column:description"`
	User          *User     `json:"user" gorm:"-" sql:"-"`
	Public        bool      `json:"-" gorm:"column:public"`
}

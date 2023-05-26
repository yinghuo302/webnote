package db

import (
	"errors"

	"github.com/google/uuid"
)

func GetArticleCount(keywd string) (int64, error) {
	var allFiles int64
	if err := dbCon.Where("description like ? and public = ?", "%"+keywd+"%", true).
		Count(&allFiles).Error; err != nil {
		return 0, nil
	}
	return allFiles, nil
}

func GetArticles(keywd string, pageNum int) ([]Files, error) {
	var ret []Files
	if err := dbCon.Limit(10).Offset((pageNum-1)*10).
		Where("description like ? and public = ?", "%"+keywd+"%", true).Find(&ret).Error; err != nil {
		return nil, nil
	}
	return ret, nil
}

func GetArticle(uuid uuid.UUID) (*Files, error) {
	ret := &Files{}
	if err := dbCon.First(ret, uuid).Error; err != nil {
		return nil, errors.New("file not found")
	}
	return ret, nil
}

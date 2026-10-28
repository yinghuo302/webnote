package db

import "github.com/google/uuid"

func GetArticleCount(keywd string) (int64, error) {
	var count int64
	err := dbCon.Model(&Files{}).Where("(name LIKE ? OR description LIKE ?) AND public = ?", "%"+keywd+"%", "%"+keywd+"%", true).Count(&count).Error
	return count, err
}
func GetArticles(keywd string, page int) ([]Files, error) {
	var results []Files
	err := dbCon.Select("user_id", "name", "file_id", "description", "public").Limit(10).Offset((page-1)*10).Where("(name LIKE ? OR description LIKE ?) AND public = ?", "%"+keywd+"%", "%"+keywd+"%", true).Order("name").Find(&results).Error
	return results, err
}
func GetArticle(id uuid.UUID) (*Files, error) {
	var result Files
	err := dbCon.Where("file_id = ? AND public = ?", id, true).First(&result).Error
	return &result, err
}

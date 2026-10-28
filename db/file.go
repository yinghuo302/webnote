package db

import (
	"errors"
	"github.com/google/uuid"
)

func GetFiles(uid int64) ([]Files, error) {
	var files []Files
	err := dbCon.Select("user_id", "name", "file_id", "description", "public").Where("user_id = ?", uid).Order("name").Find(&files).Error
	return files, err
}
func CheckFileUser(id uuid.UUID, uid int64) (*Files, error) {
	file := &Files{}
	if err := dbCon.Where("file_id = ? AND user_id = ?", id, uid).First(file).Error; err != nil {
		return nil, errors.New("文件不存在")
	}
	return file, nil
}
func SaveFile(file *Files) (uuid.UUID, error) {
	file.ContentStored = true
	if file.FileId == uuid.Nil {
		file.FileId = uuid.New()
		err := dbCon.Create(file).Error
		return file.FileId, err
	}
	err := dbCon.Model(&Files{}).Where("file_id = ? AND user_id = ?", file.FileId, file.UserId).Updates(map[string]interface{}{"name": file.Name, "description": file.Description, "content": file.Content, "content_stored": true}).Error
	return file.FileId, err
}
func ShareFile(id uuid.UUID) error {
	return dbCon.Model(&Files{}).Where("file_id = ?", id).Update("public", true).Error
}
func DeleteFile(id uuid.UUID) error { return dbCon.Where("file_id = ?", id).Delete(&Files{}).Error }

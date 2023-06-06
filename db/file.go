package db

import (
	"errors"

	"github.com/google/uuid"
)

func GetFiles(uid int64) ([]Files, error) {
	var files []Files
	if err := dbCon.Where(&Files{UserId: uid}).Find(&files).Error; err != nil {
		return nil, errors.New("文件列表获取失败")
	}
	return files, nil
}

func CheckFileUser(fileId uuid.UUID, userid int64) (*Files, error) {
	file := &Files{}
	if err := dbCon.First(file, fileId).Error; err != nil {
		return nil, errors.New("文件列表获取失败")
	}
	if file.UserId != userid {
		return nil, errors.New("the owner of file isn't you")
	}
	return file, nil
}

func SaveFile(file *Files) (uuid.UUID, error) {
	if file.FileId == uuid.Nil {
		file.FileId = uuid.New()
		if err := dbCon.Create(file).Error; err != nil {
			return uuid.Nil, err
		}
		return file.FileId, nil
	}
	if err := dbCon.Model(file).Updates(file).Error; err != nil {
		return uuid.Nil, err
	}
	return file.FileId, nil
}

func ShareFile(fileId uuid.UUID) error {
	file := &Files{FileId: fileId, Public: true}
	if err := dbCon.Model(file).Select("public").Updates(file).Error; err != nil {
		return errors.New("分享文件失败")
	}
	return nil
}

func DeleteFile(fileId uuid.UUID) error {
	if err := dbCon.Delete(&Files{FileId: fileId}).Error; err != nil {
		return errors.New("文件删除失败")
	}
	return nil
}

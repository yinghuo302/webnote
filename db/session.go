package db

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"
)

const SessionTTL = 24 * time.Hour

type Session struct {
	TokenHash string    `gorm:"primaryKey;size:64"`
	UserId    int64     `gorm:"index;not null"`
	ExpiresAt time.Time `gorm:"index;not null"`
}

func sessionHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func CreateSession(userID int64) (string, error) {
	if userID <= 0 {
		return "", errors.New("invalid user")
	}
	if err := dbCon.Where("expires_at <= ?", time.Now()).Delete(&Session{}).Error; err != nil {
		return "", err
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}
	token := hex.EncodeToString(secret)
	if err := dbCon.Create(&Session{TokenHash: sessionHash(token), UserId: userID, ExpiresAt: time.Now().Add(SessionTTL)}).Error; err != nil {
		return "", err
	}
	return token, nil
}
func LookupSession(token string) (int64, error) {
	if len(token) != 64 {
		return 0, errors.New("invalid session")
	}
	var session Session
	err := dbCon.Where("token_hash = ? AND expires_at > ?", sessionHash(token), time.Now()).First(&session).Error
	if err != nil {
		return 0, errors.New("invalid session")
	}
	return session.UserId, nil
}
func DeleteSession(token string) error {
	if len(token) != 64 {
		return nil
	}
	return dbCon.Where("token_hash = ?", sessionHash(token)).Delete(&Session{}).Error
}
func DeleteUserSessions(userID int64) error {
	return dbCon.Where("user_id = ?", userID).Delete(&Session{}).Error
}

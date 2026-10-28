package db

import (
	"context"
	"fmt"
	"time"
	"webnote/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var dbCon *gorm.DB

func InitDB() error {
	connection, err := gorm.Open(postgres.Open(config.Conf.DatabaseURL), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return fmt.Errorf("connect PostgreSQL: %w", err)
	}
	sqlDB, err := connection.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err = sqlDB.PingContext(ctx); err != nil {
		sqlDB.Close()
		return fmt.Errorf("ping PostgreSQL: %w", err)
	}
	if err = connection.AutoMigrate(&Login{}, &Code{}, &User{}, &Files{}, &Session{}); err != nil {
		sqlDB.Close()
		return fmt.Errorf("migrate PostgreSQL schema: %w", err)
	}
	dbCon = connection
	return nil
}
func GetDB() *gorm.DB { return dbCon }
func Close() error {
	if dbCon == nil {
		return nil
	}
	sqlDB, err := dbCon.DB()
	dbCon = nil
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

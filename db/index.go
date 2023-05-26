package db

import (
	"fmt"
	"log"
	"strings"
	"time"
	"webnote/config"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

var dbCon *gorm.DB

func InitDB() error {
	var err error
	dbCon, err = gorm.Open(connectDB(), initConfig())
	dbCon.AutoMigrate(&Login{})
	dbCon.AutoMigrate(&Code{})
	dbCon.AutoMigrate(&User{})
	dbCon.AutoMigrate(&Files{})
	return err
}

func initConfig() *gorm.Config {
	return &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix: config.Conf.SQLDriver.TablePrefix,
		},
		Logger: initLogger(),
	}
}

func initLogger() logger.Interface {
	logLevel := logger.Silent
	if config.Conf.Debug {
		logLevel = logger.Info
	}
	newLogger := logger.New(
		log.New(log.Writer(), "\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logLevel,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)
	return newLogger
}

func connectDB() gorm.Dialector {
	var sql gorm.Dialector
	sqlDriver := &config.Conf.SQLDriver
	switch sqlDriver.Type {
	case "sqlite":
		{
			if !(strings.HasSuffix(sqlDriver.DBFile, ".db") && len(sqlDriver.DBFile) > 3) {
				log.Fatalf("db name error.")
			}
			sql = sqlite.Open(fmt.Sprintf("%s?_journal=WAL&_vacuum=incremental", sqlDriver.DBFile))
		}
	case "mysql":
		{
			sql = mysql.Open(fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&tls=%s", sqlDriver.User, sqlDriver.Password, sqlDriver.Host, sqlDriver.Port, sqlDriver.Name, sqlDriver.SSLMode))
		}
	case "postgres":
		{
			sql = postgres.Open(fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=Asia/Shanghai", sqlDriver.Host, sqlDriver.User, sqlDriver.Password, sqlDriver.Name, sqlDriver.Port, sqlDriver.SSLMode))
		}
	default:
		log.Fatalf("not supported database type: %s", sqlDriver.Type)
	}
	return sql
}

func GetDB() *gorm.DB {
	return dbCon
}

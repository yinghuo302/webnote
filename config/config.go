package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Address       string
	Port          int
	DataDir       string
	ImgDir        string
	SecureCookies bool
	DatabaseURL   string
	Email         Email
	Debug         bool
}
type Email struct {
	Host string
	Port int
	User string
	Auth string
}

var Conf *Config

func secretValues() map[string]string {
	values := map[string]string{}
	dir := os.Getenv("CREDENTIALS_DIRECTORY")
	if dir == "" {
		return values
	}
	content, err := os.ReadFile(dir + "/webnote.env")
	if err != nil {
		return values
	}
	for _, line := range strings.Split(string(content), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if ok && key != "" && !strings.HasPrefix(key, "#") {
			values[key] = strings.TrimSuffix(value, "\r")
		}
	}
	return values
}
func envValue(key string, credentials map[string]string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return credentials[key]
}
func envInt(key string, fallback int, credentials map[string]string) (int, error) {
	value := envValue(key, credentials)
	if value == "" {
		return fallback, nil
	}
	return strconv.Atoi(value)
}
func Init() error {
	credentials := secretValues()
	port, err := envInt("WEBNOTE_PORT", 8000, credentials)
	if err != nil || port < 1 || port > 65535 {
		return errors.New("invalid WEBNOTE_PORT")
	}
	smtpPort, err := envInt("WEBNOTE_SMTP_PORT", 587, credentials)
	if err != nil || smtpPort < 1 || smtpPort > 65535 {
		return errors.New("invalid WEBNOTE_SMTP_PORT")
	}
	database := envValue("WEBNOTE_DATABASE_URL", credentials)
	if database == "" {
		return errors.New("WEBNOTE_DATABASE_URL is required (PostgreSQL connection URL)")
	}
	Conf = &Config{Address: env("WEBNOTE_ADDRESS", "127.0.0.1"), Port: port, DataDir: env("WEBNOTE_DATA_DIR", "./data"), ImgDir: env("WEBNOTE_IMG_DIR", "./img"), SecureCookies: os.Getenv("WEBNOTE_SECURE_COOKIES") != "false", DatabaseURL: database, Email: Email{Host: envValue("WEBNOTE_SMTP_HOST", credentials), Port: smtpPort, User: envValue("WEBNOTE_SMTP_USER", credentials), Auth: envValue("WEBNOTE_SMTP_PASSWORD", credentials)}, Debug: os.Getenv("WEBNOTE_DEBUG") == "true"}
	return nil
}
func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

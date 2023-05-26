package config

type Config struct {
	Address   string
	Port      int
	DataDir   string
	ImgDir    string
	Email     Email
	SQLDriver SQLConfig
	Debug     bool
}

type Email struct {
	Host string
	Port int
	User string
	Auth string
}

type SQLConfig struct {
	Type        string `json:"type" default:"sqlite"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	User        string `json:"user"`
	Password    string `json:"password"`
	Name        string `json:"name"`
	DBFile      string `json:"db_file" default:"webnote.db"`
	TablePrefix string `json:"table_prefix"`
	SSLMode     string `json:"ssl_mode"`
}

var Conf *Config

func Init() {
	Conf = &Config{
		Address: "0.0.0.0",
		Port:    8000,
		DataDir: "./data",
		ImgDir:  "./img",
		Email: Email{
			Host: "smtp.qq.com",
			Port: 587,
			User: "1150432422@qq.com",
			Auth: "oitduwsgsenefigi",
		},
		Debug: true,
		SQLDriver: SQLConfig{
			Type:   "sqlite",
			DBFile: "webnote.db",
		},
	}
}

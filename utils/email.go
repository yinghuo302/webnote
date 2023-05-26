package utils

import (
	"webnote/config"

	"gopkg.in/gomail.v2"
)

func SendMail(mailTo string, subject, body string) error {
	conf := &config.Conf.Email
	m := gomail.NewMessage()
	m.SetHeader("From", conf.User)
	m.SetHeader("To", mailTo)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)
	d := gomail.NewDialer(conf.Host, conf.Port, conf.User, conf.Auth)
	err := d.DialAndSend(m)
	return err
}

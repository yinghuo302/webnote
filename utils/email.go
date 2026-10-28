package utils

import (
	"crypto/tls"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"time"
	"webnote/config"
)

func SendMail(to, subject, body string) error {
	conf := config.Conf.Email
	if strings.ContainsAny(to+conf.User, "\r\n") {
		return errors.New("invalid mail address")
	}
	conn, err := (&net.Dialer{Timeout: 10 * time.Second}).Dial("tcp", net.JoinHostPort(conf.Host, fmt.Sprint(conf.Port)))
	if err != nil {
		return err
	}
	defer conn.Close()
	if err = conn.SetDeadline(time.Now().Add(20 * time.Second)); err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, conf.Host)
	if err != nil {
		return err
	}
	defer client.Close()
	if ok, _ := client.Extension("STARTTLS"); !ok {
		return errors.New("SMTP server must support STARTTLS")
	}
	if err = client.StartTLS(&tls.Config{ServerName: conf.Host, MinVersion: tls.VersionTLS12}); err != nil {
		return err
	}
	if err = client.Auth(smtp.PlainAuth("", conf.User, conf.Auth, conf.Host)); err != nil {
		return err
	}
	if err = client.Mail(conf.User); err != nil {
		return err
	}
	if err = client.Rcpt(to); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n", conf.User, to, mime.QEncoding.Encode("utf-8", subject), body)
	if _, err = writer.Write([]byte(message)); err != nil {
		writer.Close()
		return err
	}
	if err = writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}

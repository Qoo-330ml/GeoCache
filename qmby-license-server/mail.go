package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"strings"
)

type smtpConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

func (a *app) loadSMTPConfig() smtpConfig {
	port, _ := strconv.Atoi(a.getSetting("SMTP_PORT", "587"))
	return smtpConfig{
		Host:     strings.TrimSpace(a.getSetting("SMTP_HOST", "")),
		Port:     port,
		Username: strings.TrimSpace(a.getSetting("SMTP_USER", "")),
		Password: a.getSetting("SMTP_PASSWORD", ""),
		From:     strings.TrimSpace(a.getSetting("SMTP_FROM", "")),
	}
}

func (c smtpConfig) ready() bool {
	return c.Host != "" && c.Port > 0 && c.From != ""
}

func sendActivationEmail(cfg smtpConfig, to string, level string, code string) error {
	if !cfg.ready() {
		return fmt.Errorf("SMTP is not configured")
	}
	subject := "Qmby 激活码"
	body := fmt.Sprintf("你好，\n\n你的 Qmby %s 激活码如下：\n\n%s\n\n请在 Qmby 后台填写邮箱完成会员校验。\n", levelDisplayNames[level], code)
	message := strings.Join([]string{
		"From: " + cfg.From,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
		"",
		body,
	}, "\r\n")

	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	if cfg.Port == 465 {
		conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12})
		if err != nil {
			return err
		}
		defer conn.Close()
		client, err := smtp.NewClient(conn, cfg.Host)
		if err != nil {
			return err
		}
		defer client.Close()
		return sendMailWithClient(client, auth, cfg.From, to, []byte(message))
	}
	return smtp.SendMail(addr, auth, cfg.From, []string{to}, []byte(message))
}

func sendMailWithClient(client *smtp.Client, auth smtp.Auth, from string, to string, msg []byte) error {
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(msg); err != nil {
		_ = writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}

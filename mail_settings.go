package main

import (
	"errors"
	"net/http"
	"net/mail"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const mailSettingID = 1

type mailSettingsResponse struct {
	Host        string `json:"host"`
	Port        string `json:"port"`
	Username    string `json:"username"`
	From        string `json:"from"`
	Configured  bool   `json:"configured"`
	PasswordSet bool   `json:"password_set"`
}

func (a *app) getMailSettings(c *gin.Context) {
	setting, err := a.mailSetting()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, mailSettingsFromModel(setting))
}

func (a *app) updateMailSettings(c *gin.Context) {
	var req struct {
		Host     string `json:"host"`
		Port     string `json:"port"`
		Username string `json:"username"`
		Password string `json:"password"`
		From     string `json:"from"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}
	setting, err := a.mailSetting()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	setting.Host = strings.TrimSpace(req.Host)
	setting.Port = strings.TrimSpace(req.Port)
	setting.Username = strings.TrimSpace(req.Username)
	if strings.TrimSpace(req.Password) != "" {
		setting.Password = req.Password
	}
	setting.From = strings.TrimSpace(req.From)
	if setting.Port == "" {
		setting.Port = "587"
	}
	if setting.From != "" {
		if _, err := mail.ParseAddress(setting.From); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from address"})
			return
		}
	}
	if err := a.db.Save(&setting).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, mailSettingsFromModel(setting))
}

func (a *app) testMailSettings(c *gin.Context) {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}
	email := normalizeEmail(req.Email)
	if !validEmail(email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email"})
		return
	}
	mailer, err := a.activeMailer()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !mailer.Configured() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "mail settings are not configured"})
		return
	}
	if err := mailer.SendActivationCode(email, "QMBY-TEST-CODE", "测试邮件"); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (a *app) activeMailer() (smtpMailer, error) {
	setting, err := a.mailSetting()
	if err != nil {
		return smtpMailer{}, err
	}
	mailer := smtpMailer{
		Host:     setting.Host,
		Port:     setting.Port,
		Username: setting.Username,
		Password: setting.Password,
		From:     setting.From,
		sendMail: a.mailer.sendMail,
	}
	if mailer.Configured() {
		return mailer, nil
	}
	return a.mailer, nil
}

func (a *app) mailSetting() (MailSetting, error) {
	var setting MailSetting
	err := a.db.First(&setting, mailSettingID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return MailSetting{ID: mailSettingID, Port: "587"}, nil
	}
	return setting, err
}

func mailSettingsFromModel(setting MailSetting) mailSettingsResponse {
	mailer := smtpMailer{Host: setting.Host, Port: setting.Port, From: setting.From}
	return mailSettingsResponse{
		Host:        setting.Host,
		Port:        setting.Port,
		Username:    setting.Username,
		From:        setting.From,
		Configured:  mailer.Configured(),
		PasswordSet: strings.TrimSpace(setting.Password) != "",
	}
}

package main

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type appSettings struct {
	PublicBaseURL     string `json:"public_base_url"`
	PriceTrialCNY     string `json:"price_trial_cny"`
	PriceYearlyCNY    string `json:"price_yearly_cny"`
	PricePermanentCNY string `json:"price_permanent_cny"`
	AlipayGateway     string `json:"alipay_gateway"`
	AlipayAppID       string `json:"alipay_app_id"`
	AlipayPrivateKey  string `json:"alipay_private_key"`
	AlipayPublicKey   string `json:"alipay_public_key"`
	SMTPHost          string `json:"smtp_host"`
	SMTPPort          string `json:"smtp_port"`
	SMTPUser          string `json:"smtp_user"`
	SMTPPassword      string `json:"smtp_password"`
	SMTPFrom          string `json:"smtp_from"`
}

var settingKeys = []string{
	"PUBLIC_BASE_URL",
	"PRICE_TRIAL_CNY",
	"PRICE_YEARLY_CNY",
	"PRICE_PERMANENT_CNY",
	"ALIPAY_GATEWAY",
	"ALIPAY_APP_ID",
	"ALIPAY_PRIVATE_KEY",
	"ALIPAY_PUBLIC_KEY",
	"SMTP_HOST",
	"SMTP_PORT",
	"SMTP_USER",
	"SMTP_PASSWORD",
	"SMTP_FROM",
}

func (a *app) getSetting(key string, fallback string) string {
	var setting Setting
	if err := a.db.First(&setting, "key = ?", key).Error; err == nil && strings.TrimSpace(setting.Value) != "" {
		return setting.Value
	}
	return env(key, fallback)
}

func (a *app) currentSettings() appSettings {
	return appSettings{
		PublicBaseURL:     a.getSetting("PUBLIC_BASE_URL", "http://localhost:2090"),
		PriceTrialCNY:     a.getSetting("PRICE_TRIAL_CNY", "0.00"),
		PriceYearlyCNY:    a.getSetting("PRICE_YEARLY_CNY", "8.80"),
		PricePermanentCNY: a.getSetting("PRICE_PERMANENT_CNY", "18.80"),
		AlipayGateway:     a.getSetting("ALIPAY_GATEWAY", "https://openapi.alipay.com/gateway.do"),
		AlipayAppID:       a.getSetting("ALIPAY_APP_ID", ""),
		AlipayPrivateKey:  a.getSetting("ALIPAY_PRIVATE_KEY", ""),
		AlipayPublicKey:   a.getSetting("ALIPAY_PUBLIC_KEY", ""),
		SMTPHost:          a.getSetting("SMTP_HOST", ""),
		SMTPPort:          a.getSetting("SMTP_PORT", "587"),
		SMTPUser:          a.getSetting("SMTP_USER", ""),
		SMTPPassword:      a.getSetting("SMTP_PASSWORD", ""),
		SMTPFrom:          a.getSetting("SMTP_FROM", ""),
	}
}

func (a *app) getSettings(c *gin.Context) {
	c.JSON(http.StatusOK, a.currentSettings())
}

func (a *app) putSettings(c *gin.Context) {
	var req appSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}
	if _, err := strconv.Atoi(strings.TrimSpace(req.SMTPPort)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "SMTP port must be a number"})
		return
	}
	values := map[string]string{
		"PUBLIC_BASE_URL":     req.PublicBaseURL,
		"PRICE_TRIAL_CNY":     req.PriceTrialCNY,
		"PRICE_YEARLY_CNY":    req.PriceYearlyCNY,
		"PRICE_PERMANENT_CNY": req.PricePermanentCNY,
		"ALIPAY_GATEWAY":      req.AlipayGateway,
		"ALIPAY_APP_ID":       req.AlipayAppID,
		"ALIPAY_PRIVATE_KEY":  req.AlipayPrivateKey,
		"ALIPAY_PUBLIC_KEY":   req.AlipayPublicKey,
		"SMTP_HOST":           req.SMTPHost,
		"SMTP_PORT":           req.SMTPPort,
		"SMTP_USER":           req.SMTPUser,
		"SMTP_PASSWORD":       req.SMTPPassword,
		"SMTP_FROM":           req.SMTPFrom,
	}
	if err := a.db.Transaction(func(tx *gorm.DB) error {
		for _, key := range settingKeys {
			setting := Setting{Key: key, Value: strings.TrimSpace(values[key])}
			if key == "ALIPAY_PRIVATE_KEY" || key == "ALIPAY_PUBLIC_KEY" || key == "SMTP_PASSWORD" {
				setting.Value = values[key]
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "key"}},
				DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
			}).Create(&setting).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, a.currentSettings())
}

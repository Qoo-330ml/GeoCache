package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type product struct {
	Level       string `json:"level"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Amount      string `json:"amount"`
}

func (a *app) publicProducts() []product {
	return []product{
		{Level: LevelTrial, Label: "7天试用", Description: "适合先体验会员功能", Amount: a.getSetting("PRICE_TRIAL_CNY", "0.00")},
		{Level: LevelYearly, Label: "年费会员", Description: "从首次联网激活开始计算 365 天", Amount: a.getSetting("PRICE_YEARLY_CNY", "8.80")},
		{Level: LevelPermanent, Label: "永久会员", Description: "一次购买，长期有效", Amount: a.getSetting("PRICE_PERMANENT_CNY", "18.80")},
	}
}

func (a *app) productByLevel(level string) (product, bool) {
	for _, item := range a.publicProducts() {
		if item.Level == level {
			return item, true
		}
	}
	return product{}, false
}

func serveBuy(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, buyHTML)
}

func servePayReturn(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, payReturnHTML)
}

func (a *app) listProducts(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"products": a.publicProducts()})
}

func (a *app) createPurchaseOrder(c *gin.Context) {
	var req struct {
		Email string `json:"email"`
		Level string `json:"level"`
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
	level := strings.ToLower(strings.TrimSpace(req.Level))
	item, ok := a.productByLevel(level)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product"})
		return
	}
	if _, err := strconv.ParseFloat(item.Amount, 64); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid product amount"})
		return
	}

	order := PurchaseOrder{
		OrderNo: generateOrderNo(),
		Email:   email,
		Level:   level,
		Amount:  item.Amount,
		Status:  OrderStatusPending,
	}
	if err := a.db.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	cfg := a.loadAlipayConfig()
	if !cfg.ready() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":    "alipay is not configured",
			"order_no": order.OrderNo,
		})
		return
	}

	formHTML, err := buildAlipayPagePayForm(cfg, a.getSetting("PUBLIC_BASE_URL", "http://localhost:2090"), order, "Qmby "+item.Label)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"order_no": order.OrderNo, "pay_form_html": formHTML})
}

func (a *app) alipayNotify(c *gin.Context) {
	if err := c.Request.ParseForm(); err != nil {
		c.String(http.StatusBadRequest, "fail")
		return
	}
	params := make(map[string]string, len(c.Request.PostForm))
	for key, values := range c.Request.PostForm {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}
	cfg := a.loadAlipayConfig()
	if !verifyAlipayParams(copyMap(params), cfg.PublicKey) {
		c.String(http.StatusBadRequest, "fail")
		return
	}
	if params["app_id"] != cfg.AppID {
		c.String(http.StatusBadRequest, "fail")
		return
	}
	tradeStatus := params["trade_status"]
	if tradeStatus != "TRADE_SUCCESS" && tradeStatus != "TRADE_FINISHED" {
		c.String(http.StatusOK, "success")
		return
	}

	if err := a.fulfillPaidOrder(params["out_trade_no"], params["trade_no"], params["buyer_logon_id"]); err != nil {
		c.String(http.StatusInternalServerError, "fail")
		return
	}
	c.String(http.StatusOK, "success")
}

func (a *app) fulfillPaidOrder(orderNo string, tradeNo string, buyerLogonID string) error {
	now := time.Now().In(beijingLocation())
	var order PurchaseOrder
	var plainCode string

	err := a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("order_no = ?", orderNo).First(&order).Error; err != nil {
			return err
		}
		if order.Status == OrderStatusFulfilled {
			return nil
		}
		order.Status = OrderStatusPaid
		order.AlipayTradeNo = tradeNo
		order.BuyerLogonID = buyerLogonID
		order.PaidAt = &now

		if order.ActivationCodeID == nil {
			var err error
			plainCode, err = generateCode()
			if err != nil {
				return err
			}
			code := ActivationCode{
				CodeHash:     hashCode(plainCode),
				CodePrefix:   codePrefix(plainCode),
				Email:        order.Email,
				Level:        order.Level,
				DurationDays: levelDurationDays[order.Level],
				Status:       StatusIssued,
				Note:         "purchase order " + order.OrderNo,
			}
			if err := tx.Create(&code).Error; err != nil {
				return err
			}
			order.ActivationCodeID = &code.ID
			order.ActivationCodePlain = plainCode
		} else {
			plainCode = order.ActivationCodePlain
		}
		if plainCode == "" {
			return fmt.Errorf("activation code plaintext is missing")
		}
		if err := tx.Save(&order).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	if order.Status == OrderStatusFulfilled {
		return nil
	}

	if err := sendActivationEmail(a.loadSMTPConfig(), order.Email, order.Level, plainCode); err != nil {
		a.db.Model(&PurchaseOrder{}).Where("order_no = ?", orderNo).Updates(map[string]any{
			"status":     OrderStatusEmailFailed,
			"last_error": err.Error(),
		})
		return err
	}
	return a.db.Model(&PurchaseOrder{}).Where("order_no = ?", orderNo).Updates(map[string]any{
		"status":        OrderStatusFulfilled,
		"fulfilled_at":  now,
		"email_sent_at": now,
		"last_error":    "",
	}).Error
}

func generateOrderNo() string {
	code, err := generateCode()
	if err != nil {
		return fmt.Sprintf("QMBY%d", time.Now().UnixNano())
	}
	code = strings.ReplaceAll(code, "-", "")
	return "QMBY" + time.Now().Format("20060102150405") + code[len(code)-8:]
}

func copyMap(src map[string]string) map[string]string {
	dst := make(map[string]string, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

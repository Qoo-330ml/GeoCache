package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	paymentSettingID       = 1
	xorpayProvider         = "xorpay"
	defaultXorPayGateway   = "https://xorpay.com"
	defaultOrderExpireSecs = 7200
)

type billingPlan struct {
	Key          string `json:"key"`
	Name         string `json:"name"`
	Level        string `json:"level"`
	DurationDays int    `json:"duration_days"`
	AmountCents  int    `json:"amount_cents"`
	PriceLabel   string `json:"price_label"`
	Description  string `json:"description"`
}

var billingPlans = []billingPlan{
	{Key: "plus_year", Name: "Plus 年费", Level: LevelPlus, DurationDays: 365, AmountCents: 1880, PriceLabel: "¥18.8 / 年", Description: "保障非大规模开服用户的正常使用"},
	{Key: "plus_lifetime", Name: "Plus 永久", Level: LevelPlus, DurationDays: 0, AmountCents: 5880, PriceLabel: "¥58.8 / 永久", Description: "Plus 长期授权"},
	{Key: "pro_year", Name: "Pro 年费", Level: LevelPro, DurationDays: 365, AmountCents: 4880, PriceLabel: "¥48.8 / 年", Description: "解锁所有功能和开服专项能力"},
	{Key: "pro_lifetime", Name: "Pro 永久", Level: LevelPro, DurationDays: 0, AmountCents: 19880, PriceLabel: "¥198.8 / 永久", Description: "Pro 长期授权"},
}

type paymentSettingsResponse struct {
	PublicBaseURL      string   `json:"public_base_url"`
	XorPayAID          string   `json:"xorpay_aid"`
	XorPayConfigured   bool     `json:"xorpay_configured"`
	EnabledPayTypes    []string `json:"enabled_pay_types"`
	OrderExpireSeconds int      `json:"order_expire_seconds"`
}

type paymentOrderResponse struct {
	OrderID         string     `json:"order_id"`
	Provider        string     `json:"provider"`
	PayType         string     `json:"pay_type"`
	PlanKey         string     `json:"plan_key"`
	PlanLevel       string     `json:"plan_level"`
	DurationDays    int        `json:"duration_days"`
	AmountCents     int        `json:"amount_cents"`
	Status          string     `json:"status"`
	Email           string     `json:"email"`
	InstanceID      string     `json:"instance_id"`
	PayQRCode       string     `json:"pay_qr,omitempty"`
	ActivationCode  string     `json:"activation_code,omitempty"`
	ProviderOrderID string     `json:"provider_order_id,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	PaidAt          *time.Time `json:"paid_at,omitempty"`
	ExpiresAt       time.Time  `json:"expires_at"`
	LastError       string     `json:"last_error,omitempty"`
}

type xorpayCreateResponse struct {
	OK              bool
	ProviderOrderID string
	QR              string
	Raw             string
	Message         string
}

func (a *app) listBillingPlans(c *gin.Context) {
	settings, _ := a.paymentSetting()
	c.JSON(http.StatusOK, gin.H{
		"plans":             billingPlans,
		"enabled_pay_types": enabledPayTypes(settings.EnabledPayTypes),
	})
}

func (a *app) createBillingOrder(c *gin.Context) {
	var req struct {
		Email       string `json:"email"`
		InstanceID  string `json:"instance_id"`
		PlanKey     string `json:"plan_key"`
		PayType     string `json:"pay_type"`
		QmbyVersion string `json:"qmby_version"`
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
	instanceID := strings.TrimSpace(req.InstanceID)
	if instanceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "instance_id required"})
		return
	}
	plan, ok := billingPlanByKey(req.PlanKey)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan"})
		return
	}
	payType := strings.ToLower(strings.TrimSpace(req.PayType))
	if payType != "alipay" && payType != "native" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pay_type"})
		return
	}
	settings, err := a.paymentSetting()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !paymentTypeEnabled(settings.EnabledPayTypes, payType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pay_type is disabled"})
		return
	}
	if !xorpayReady(settings) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "xorpay is not configured"})
		return
	}
	if strings.TrimSpace(settings.PublicBaseURL) == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "public_base_url is not configured"})
		return
	}

	var existing int64
	if err := a.db.Model(&ActivationCode{}).
		Where("email = ? AND status IN ?", email, []string{StatusIssued, StatusActive}).
		Count(&existing).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if existing > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "email already has an available license"})
		return
	}

	order := PaymentOrder{
		OrderID:      generateBillingOrderID(),
		Provider:     xorpayProvider,
		PayType:      payType,
		PlanKey:      plan.Key,
		PlanLevel:    plan.Level,
		DurationDays: plan.DurationDays,
		AmountCents:  plan.AmountCents,
		Status:       OrderStatusPending,
		Email:        email,
		InstanceID:   instanceID,
		ExpiresAt:    time.Now().In(beijingLocation()).Add(time.Duration(orderExpireSeconds(settings)) * time.Second),
	}

	createResp, err := a.createXorPayOrder(settings, order, plan)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	order.ProviderOrderID = createResp.ProviderOrderID
	order.ProviderPayload = createResp.Raw
	order.PayQRCode = createResp.QR
	if createResp.Message != "" && !createResp.OK {
		order.Status = OrderStatusFailed
		order.LastError = createResp.Message
	}
	if err := a.db.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !createResp.OK || strings.TrimSpace(createResp.QR) == "" {
		c.JSON(http.StatusBadGateway, gin.H{"error": firstNonEmpty(createResp.Message, "create xorpay order failed"), "order": paymentOrderPayload(order, false)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"order": paymentOrderPayload(order, false)})
}

func (a *app) getBillingOrder(c *gin.Context) {
	orderID := strings.TrimSpace(c.Param("order_id"))
	var order PaymentOrder
	if err := a.db.Where("order_id = ?", orderID).First(&order).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": "order not found"})
		return
	}
	a.expireOrderIfNeeded(&order)
	c.JSON(http.StatusOK, gin.H{"order": paymentOrderPayload(order, true)})
}

func (a *app) xorpayNotify(c *gin.Context) {
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
	settings, err := a.paymentSetting()
	if err != nil || !verifyXorPayNotify(params, settings.XorPaySecret) {
		c.String(http.StatusBadRequest, "fail")
		return
	}
	orderID := strings.TrimSpace(params["order_id"])
	payPrice := strings.TrimSpace(params["pay_price"])
	payTime := strings.TrimSpace(params["pay_time"])
	providerOrderID := strings.TrimSpace(params["aoid"])
	if err := a.fulfillXorPayOrder(orderID, providerOrderID, payPrice, payTime, params); err != nil {
		c.String(http.StatusInternalServerError, "fail")
		return
	}
	c.String(http.StatusOK, "success")
}

func (a *app) getPaymentSettings(c *gin.Context) {
	setting, err := a.paymentSetting()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, paymentSettingsFromModel(setting))
}

func (a *app) updatePaymentSettings(c *gin.Context) {
	var req struct {
		PublicBaseURL      string   `json:"public_base_url"`
		XorPayAID          string   `json:"xorpay_aid"`
		XorPaySecret       string   `json:"xorpay_secret"`
		EnabledPayTypes    []string `json:"enabled_pay_types"`
		OrderExpireSeconds int      `json:"order_expire_seconds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}
	setting, err := a.paymentSetting()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	setting.PublicBaseURL = strings.TrimRight(strings.TrimSpace(req.PublicBaseURL), "/")
	setting.XorPayAID = strings.TrimSpace(req.XorPayAID)
	if strings.TrimSpace(req.XorPaySecret) != "" {
		setting.XorPaySecret = strings.TrimSpace(req.XorPaySecret)
	}
	setting.EnabledPayTypes = strings.Join(cleanPayTypes(req.EnabledPayTypes), ",")
	if req.OrderExpireSeconds > 0 {
		setting.OrderExpireSeconds = req.OrderExpireSeconds
	}
	if setting.OrderExpireSeconds <= 0 {
		setting.OrderExpireSeconds = defaultOrderExpireSecs
	}
	if err := a.db.Save(&setting).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, paymentSettingsFromModel(setting))
}

func (a *app) listPaymentOrders(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	tx := a.db.Model(&PaymentOrder{})
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		tx = tx.Where("status = ?", status)
	}
	if search := strings.TrimSpace(c.Query("search")); search != "" {
		like := "%" + strings.ToLower(search) + "%"
		tx = tx.Where("LOWER(email) LIKE ? OR LOWER(order_id) LIKE ? OR LOWER(provider_order_id) LIKE ?", like, like, like)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var orders []PaymentOrder
	if err := tx.Order("created_at DESC, id DESC").Limit(limit).Offset(offset).Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	payload := make([]paymentOrderResponse, 0, len(orders))
	for _, order := range orders {
		payload = append(payload, paymentOrderPayload(order, false))
	}
	c.JSON(http.StatusOK, gin.H{"orders": payload, "total": total})
}

func (a *app) createXorPayOrder(settings PaymentSetting, order PaymentOrder, plan billingPlan) (xorpayCreateResponse, error) {
	values := url.Values{}
	values.Set("name", "Qmby "+plan.Name)
	values.Set("pay_type", order.PayType)
	values.Set("price", amountYuan(order.AmountCents))
	values.Set("order_id", order.OrderID)
	values.Set("notify_url", strings.TrimRight(settings.PublicBaseURL, "/")+"/api/pay/xorpay/notify")
	values.Set("sign", xorpayCreateSign(values.Get("name"), values.Get("pay_type"), values.Get("price"), values.Get("order_id"), values.Get("notify_url"), settings.XorPaySecret))

	endpoint := strings.TrimRight(env("XORPAY_GATEWAY", defaultXorPayGateway), "/") + "/api/pay/" + url.PathEscape(settings.XorPayAID)
	client := a.xorpayClient()
	resp, err := client.PostForm(endpoint, values)
	if err != nil {
		return xorpayCreateResponse{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	raw := string(body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return xorpayCreateResponse{Raw: raw}, fmt.Errorf("xorpay HTTP %d", resp.StatusCode)
	}
	return parseXorPayCreateResponse(raw), nil
}

func (a *app) fulfillXorPayOrder(orderID, providerOrderID, payPrice, payTime string, params map[string]string) error {
	now := time.Now().In(beijingLocation())
	var order PaymentOrder
	var plainCode string
	var shouldSendMail bool

	rawPayload, _ := json.Marshal(params)
	err := a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("order_id = ?", orderID).First(&order).Error; err != nil {
			return err
		}
		if order.Status == OrderStatusFulfilled || order.Status == OrderStatusActivated {
			return nil
		}
		if order.Status == OrderStatusExpired || now.After(order.ExpiresAt) {
			order.Status = OrderStatusExpired
			order.LastError = "order expired"
			order.ProviderPayload = string(rawPayload)
			return tx.Save(&order).Error
		}
		callbackAmountCents, err := amountCentsFromYuan(payPrice)
		if err != nil || callbackAmountCents != order.AmountCents {
			order.Status = OrderStatusFailed
			order.LastError = "amount mismatch"
			order.ProviderPayload = string(rawPayload)
			return tx.Save(&order).Error
		}

		order.Status = OrderStatusPaid
		order.ProviderOrderID = providerOrderID
		order.ProviderPayload = string(rawPayload)
		order.PaidAt = parseXorPayTime(payTime, now)

		if order.ActivationCodeID == nil {
			var existing int64
			if err := tx.Model(&ActivationCode{}).
				Where("email = ? AND status IN ?", order.Email, []string{StatusIssued, StatusActive}).
				Count(&existing).Error; err != nil {
				return err
			}
			if existing > 0 {
				order.Status = OrderStatusFailed
				order.LastError = "email already has an available license"
				return tx.Save(&order).Error
			}

			var err error
			plainCode, err = generateCode()
			if err != nil {
				return err
			}
			code := ActivationCode{
				CodeHash:     hashCode(plainCode),
				PlainCode:    plainCode,
				CodePrefix:   codePrefix(plainCode),
				Email:        order.Email,
				Level:        order.PlanLevel,
				DurationDays: order.DurationDays,
				Status:       StatusIssued,
				Note:         "billing_order:" + order.OrderID,
			}
			if err := tx.Create(&code).Error; err != nil {
				return err
			}
			order.ActivationCodeID = &code.ID
			order.ActivationCodePlain = plainCode
			shouldSendMail = true
		} else {
			plainCode = order.ActivationCodePlain
		}
		if plainCode == "" {
			return errors.New("activation code plaintext is missing")
		}
		order.Status = OrderStatusFulfilled
		order.FulfilledAt = &now
		order.LastError = ""
		return tx.Save(&order).Error
	})
	if err != nil {
		return err
	}
	if shouldSendMail {
		if err := a.sendOrderActivationEmail(order, plainCode, now); err != nil {
			a.db.Model(&PaymentOrder{}).Where("order_id = ?", orderID).Update("last_error", "send activation email: "+truncate(err.Error(), 460))
		}
	}
	return nil
}

func (a *app) sendOrderActivationEmail(order PaymentOrder, plainCode string, now time.Time) error {
	mailer, err := a.activeMailer()
	if err != nil || !mailer.Configured() {
		return err
	}
	if err := mailer.SendActivationCode(order.Email, plainCode, levelDisplayNames[order.PlanLevel]); err != nil {
		return err
	}
	return a.db.Model(&PaymentOrder{}).Where("order_id = ?", order.OrderID).Update("email_sent_at", now).Error
}

func (a *app) paymentSetting() (PaymentSetting, error) {
	var setting PaymentSetting
	err := a.db.First(&setting, paymentSettingID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return PaymentSetting{
			ID:                 paymentSettingID,
			PublicBaseURL:      strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL")), "/"),
			XorPayAID:          strings.TrimSpace(os.Getenv("XORPAY_AID")),
			XorPaySecret:       strings.TrimSpace(os.Getenv("XORPAY_SECRET")),
			EnabledPayTypes:    "alipay,native",
			OrderExpireSeconds: defaultOrderExpireSecs,
		}, nil
	}
	return setting, err
}

func paymentSettingsFromModel(setting PaymentSetting) paymentSettingsResponse {
	return paymentSettingsResponse{
		PublicBaseURL:      setting.PublicBaseURL,
		XorPayAID:          setting.XorPayAID,
		XorPayConfigured:   xorpayReady(setting),
		EnabledPayTypes:    enabledPayTypes(setting.EnabledPayTypes),
		OrderExpireSeconds: orderExpireSeconds(setting),
	}
}

func paymentOrderPayload(order PaymentOrder, includeActivationCode bool) paymentOrderResponse {
	payload := paymentOrderResponse{
		OrderID:         order.OrderID,
		Provider:        order.Provider,
		PayType:         order.PayType,
		PlanKey:         order.PlanKey,
		PlanLevel:       order.PlanLevel,
		DurationDays:    order.DurationDays,
		AmountCents:     order.AmountCents,
		Status:          order.Status,
		Email:           order.Email,
		InstanceID:      order.InstanceID,
		PayQRCode:       order.PayQRCode,
		ProviderOrderID: order.ProviderOrderID,
		CreatedAt:       order.CreatedAt,
		PaidAt:          order.PaidAt,
		ExpiresAt:       order.ExpiresAt,
		LastError:       order.LastError,
	}
	if includeActivationCode && (order.Status == OrderStatusFulfilled || order.Status == OrderStatusActivated) {
		payload.ActivationCode = order.ActivationCodePlain
	}
	return payload
}

func (a *app) expireOrderIfNeeded(order *PaymentOrder) {
	if order.Status != OrderStatusPending || time.Now().In(beijingLocation()).Before(order.ExpiresAt) {
		return
	}
	order.Status = OrderStatusExpired
	order.LastError = "order expired"
	_ = a.db.Save(order).Error
}

func (a *app) xorpayClient() *http.Client {
	if a.xorpayHTTPClient != nil {
		return a.xorpayHTTPClient
	}
	return &http.Client{Timeout: 12 * time.Second}
}

func billingPlanByKey(key string) (billingPlan, bool) {
	key = strings.TrimSpace(key)
	for _, plan := range billingPlans {
		if plan.Key == key {
			return plan, true
		}
	}
	return billingPlan{}, false
}

func xorpayReady(setting PaymentSetting) bool {
	return strings.TrimSpace(setting.XorPayAID) != "" && strings.TrimSpace(setting.XorPaySecret) != ""
}

func orderExpireSeconds(setting PaymentSetting) int {
	if setting.OrderExpireSeconds > 0 {
		return setting.OrderExpireSeconds
	}
	return defaultOrderExpireSecs
}

func cleanPayTypes(values []string) []string {
	seen := map[string]bool{}
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if (value == "alipay" || value == "native") && !seen[value] {
			cleaned = append(cleaned, value)
			seen[value] = true
		}
	}
	if len(cleaned) == 0 {
		return []string{"alipay", "native"}
	}
	return cleaned
}

func enabledPayTypes(value string) []string {
	return cleanPayTypes(strings.Split(value, ","))
}

func paymentTypeEnabled(value, payType string) bool {
	for _, item := range enabledPayTypes(value) {
		if item == payType {
			return true
		}
	}
	return false
}

func amountYuan(cents int) string {
	return fmt.Sprintf("%.2f", float64(cents)/100)
}

func amountCentsFromYuan(value string) (int, error) {
	amount, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0, err
	}
	return int(amount*100 + 0.5), nil
}

func xorpayCreateSign(name, payType, price, orderID, notifyURL, secret string) string {
	return md5Hex(name + payType + price + orderID + notifyURL + secret)
}

func verifyXorPayNotify(params map[string]string, secret string) bool {
	sign := strings.ToLower(strings.TrimSpace(params["sign"]))
	if sign == "" || strings.TrimSpace(secret) == "" {
		return false
	}
	expected := md5Hex(params["aoid"] + params["order_id"] + params["pay_price"] + params["pay_time"] + secret)
	return sign == expected
}

func md5Hex(value string) string {
	sum := md5.Sum([]byte(value))
	return hex.EncodeToString(sum[:])
}

func parseXorPayCreateResponse(raw string) xorpayCreateResponse {
	var data map[string]any
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return xorpayCreateResponse{Raw: raw, Message: "invalid xorpay response"}
	}
	lower := make(map[string]any, len(data))
	for key, value := range data {
		lower[strings.ToLower(key)] = value
	}
	message := firstNonEmpty(stringField(lower, "msg"), stringField(lower, "message"), stringField(lower, "info"), stringField(lower, "error"))
	status := strings.ToLower(firstNonEmpty(stringField(lower, "status"), stringField(lower, "code")))
	ok := status == "ok" || status == "success" || status == "1" || status == "200"
	qr := firstNonEmpty(
		stringField(lower, "qr"),
		stringField(lower, "qrcode"),
		stringField(lower, "qr_url"),
		stringField(lower, "payurl"),
		stringField(lower, "pay_url"),
		stringField(lower, "url"),
	)
	aoid := firstNonEmpty(stringField(lower, "aoid"), stringField(lower, "id"), stringField(lower, "trade_no"))
	if qr != "" && status == "" && message == "" {
		ok = true
	}
	return xorpayCreateResponse{OK: ok, ProviderOrderID: aoid, QR: qr, Raw: raw, Message: message}
}

func stringField(values map[string]any, key string) string {
	value, ok := values[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		if typed == float64(int64(typed)) {
			return strconv.FormatInt(int64(typed), 10)
		}
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func parseXorPayTime(value string, fallback time.Time) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return &fallback
	}
	loc := beijingLocation()
	layouts := []string{"2006-01-02 15:04:05", time.RFC3339}
	for _, layout := range layouts {
		if parsed, err := time.ParseInLocation(layout, value, loc); err == nil {
			return &parsed
		}
	}
	return &fallback
}

func generateBillingOrderID() string {
	code, err := generateCode()
	if err != nil {
		return fmt.Sprintf("QMBY%d", time.Now().UnixNano())
	}
	code = strings.ReplaceAll(code, "-", "")
	if len(code) > 10 {
		code = code[len(code)-10:]
	}
	return "QMBY" + time.Now().In(beijingLocation()).Format("20060102150405") + code
}

func sortedParamKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

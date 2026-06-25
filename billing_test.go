package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestCreateBillingOrderSendsSignedXorPayRequest(t *testing.T) {
	a, _ := testLicenseApp(t)
	if err := a.db.Create(&PaymentSetting{
		ID:                 paymentSettingID,
		PublicBaseURL:      "https://license.example.com",
		XorPayAID:          "aid123",
		XorPaySecret:       "secret123",
		EnabledPayTypes:    "alipay,native",
		OrderExpireSeconds: 7200,
	}).Error; err != nil {
		t.Fatalf("create payment setting: %v", err)
	}

	var got url.Values
	xorpay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/pay/aid123" {
			t.Fatalf("xorpay path = %q", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		got = r.PostForm
		expectedSign := xorpayCreateSign(
			got.Get("name"),
			got.Get("pay_type"),
			got.Get("price"),
			got.Get("order_id"),
			got.Get("notify_url"),
			"secret123",
		)
		if got.Get("sign") != expectedSign {
			t.Fatalf("sign = %q, want %q", got.Get("sign"), expectedSign)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
			"info": map[string]any{
				"qr": "https://xorpay.example/qr.png",
			},
			"aoid": "xo-1001",
		})
	}))
	defer xorpay.Close()
	t.Setenv("XORPAY_GATEWAY", xorpay.URL)

	res := postBillingOrder(t, a, `{"email":"User@Example.com","instance_id":"qmby-instance","plan_key":"plus_year","pay_type":"alipay"}`)
	if res.Code != http.StatusOK {
		t.Fatalf("create billing order status = %d, body = %s", res.Code, res.Body.String())
	}
	if got.Get("price") != "18.80" || got.Get("notify_url") != "https://license.example.com/api/pay/xorpay/notify" {
		t.Fatalf("unexpected xorpay request: %v", got)
	}

	var order PaymentOrder
	if err := a.db.Where("email = ?", "user@example.com").First(&order).Error; err != nil {
		t.Fatalf("load order: %v", err)
	}
	if order.Status != OrderStatusPending || order.PayQRCode == "" || order.ProviderOrderID != "xo-1001" {
		t.Fatalf("unexpected order: %+v", order)
	}
}

func TestXorPayNotifyFulfillsOrderIdempotently(t *testing.T) {
	a, _ := testLicenseApp(t)
	if err := a.db.Create(&PaymentSetting{
		ID:                 paymentSettingID,
		PublicBaseURL:      "https://license.example.com",
		XorPayAID:          "aid123",
		XorPaySecret:       "secret123",
		EnabledPayTypes:    "alipay,native",
		OrderExpireSeconds: 7200,
	}).Error; err != nil {
		t.Fatalf("create payment setting: %v", err)
	}
	order := PaymentOrder{
		OrderID:      "QMBY-ORDER-1",
		Provider:     xorpayProvider,
		PayType:      "native",
		PlanKey:      "pro_lifetime",
		PlanLevel:    LevelPro,
		DurationDays: 0,
		AmountCents:  19880,
		Status:       OrderStatusPending,
		Email:        "paid@example.com",
		InstanceID:   "qmby-paid-instance",
		ExpiresAt:    time.Now().In(beijingLocation()).Add(2 * time.Hour),
	}
	if err := a.db.Create(&order).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	form := url.Values{
		"aoid":      {"xo-paid-1"},
		"order_id":  {"QMBY-ORDER-1"},
		"pay_price": {"198.8"},
		"pay_time":  {"2026-06-24 12:30:00"},
	}
	form.Set("sign", md5Hex(form.Get("aoid")+form.Get("order_id")+form.Get("pay_price")+form.Get("pay_time")+"secret123"))

	for i := 0; i < 2; i++ {
		res := postXorPayNotify(t, a, form)
		if res.Code != http.StatusOK || strings.TrimSpace(res.Body.String()) != "success" {
			t.Fatalf("notify %d status = %d, body = %s", i+1, res.Code, res.Body.String())
		}
	}

	var fulfilled PaymentOrder
	if err := a.db.Where("order_id = ?", "QMBY-ORDER-1").First(&fulfilled).Error; err != nil {
		t.Fatalf("load fulfilled order: %v", err)
	}
	if fulfilled.Status != OrderStatusFulfilled || fulfilled.ActivationCodeID == nil || fulfilled.ActivationCodePlain == "" {
		t.Fatalf("unexpected fulfilled order: %+v", fulfilled)
	}
	var codes int64
	if err := a.db.Model(&ActivationCode{}).Where("email = ?", "paid@example.com").Count(&codes).Error; err != nil {
		t.Fatalf("count codes: %v", err)
	}
	if codes != 1 {
		t.Fatalf("activation code count = %d, want 1", codes)
	}
}

func postBillingOrder(t *testing.T, a *app, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := gin.New()
	r.POST("/api/billing/orders", a.requireLicenseKey(), a.createBillingOrder)
	req := httptest.NewRequest(http.MethodPost, "/api/billing/orders", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-key")
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	return res
}

func postXorPayNotify(t *testing.T, a *app, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	r := gin.New()
	r.POST("/api/pay/xorpay/notify", a.xorpayNotify)
	req := httptest.NewRequest(http.MethodPost, "/api/pay/xorpay/notify", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	return res
}

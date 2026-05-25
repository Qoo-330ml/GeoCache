package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

type alipayConfig struct {
	AppID      string
	PrivateKey string
	PublicKey  string
	Gateway    string
}

func (a *app) loadAlipayConfig() alipayConfig {
	return alipayConfig{
		AppID:      strings.TrimSpace(a.getSetting("ALIPAY_APP_ID", "")),
		PrivateKey: strings.TrimSpace(a.getSetting("ALIPAY_PRIVATE_KEY", "")),
		PublicKey:  strings.TrimSpace(a.getSetting("ALIPAY_PUBLIC_KEY", "")),
		Gateway:    a.getSetting("ALIPAY_GATEWAY", "https://openapi.alipay.com/gateway.do"),
	}
}

func (c alipayConfig) ready() bool {
	return c.AppID != "" && c.PrivateKey != ""
}

func buildAlipayPagePayForm(cfg alipayConfig, publicBaseURL string, order PurchaseOrder, subject string) (string, error) {
	privateKey, err := parseRSAPrivateKey(cfg.PrivateKey)
	if err != nil {
		return "", err
	}
	bizContent := fmt.Sprintf(`{"out_trade_no":"%s","product_code":"FAST_INSTANT_TRADE_PAY","total_amount":"%s","subject":"%s"}`,
		escapeJSON(order.OrderNo), escapeJSON(order.Amount), escapeJSON(subject))
	params := map[string]string{
		"app_id":      cfg.AppID,
		"method":      "alipay.trade.page.pay",
		"format":      "JSON",
		"charset":     "utf-8",
		"sign_type":   "RSA2",
		"timestamp":   time.Now().In(beijingLocation()).Format("2006-01-02 15:04:05"),
		"version":     "1.0",
		"notify_url":  strings.TrimRight(publicBaseURL, "/") + "/api/pay/alipay/notify",
		"return_url":  strings.TrimRight(publicBaseURL, "/") + "/pay/return?order_no=" + url.QueryEscape(order.OrderNo),
		"biz_content": bizContent,
	}
	sign, err := signAlipayParams(params, privateKey)
	if err != nil {
		return "", err
	}
	params["sign"] = sign

	var b strings.Builder
	b.WriteString(`<!doctype html><html><head><meta charset="utf-8"><title>正在跳转支付宝</title></head><body>`)
	b.WriteString(`<form id="pay" method="post" action="`)
	b.WriteString(htmlEscape(cfg.Gateway))
	b.WriteString(`">`)
	keys := sortedKeys(params)
	for _, key := range keys {
		b.WriteString(`<input type="hidden" name="`)
		b.WriteString(htmlEscape(key))
		b.WriteString(`" value="`)
		b.WriteString(htmlEscape(params[key]))
		b.WriteString(`">`)
	}
	b.WriteString(`</form><script>document.getElementById("pay").submit();</script></body></html>`)
	return b.String(), nil
}

func signAlipayParams(params map[string]string, privateKey *rsa.PrivateKey) (string, error) {
	payload := canonicalAlipayParams(params)
	sum := sha256.Sum256([]byte(payload))
	sig, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, sum[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

func verifyAlipayParams(params map[string]string, publicKeyText string) bool {
	signature := params["sign"]
	if signature == "" || strings.TrimSpace(publicKeyText) == "" {
		return false
	}
	publicKey, err := parseRSAPublicKey(publicKeyText)
	if err != nil {
		return false
	}
	sig, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false
	}
	delete(params, "sign")
	delete(params, "sign_type")
	payload := canonicalAlipayParams(params)
	sum := sha256.Sum256([]byte(payload))
	return rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, sum[:], sig) == nil
}

func canonicalAlipayParams(params map[string]string) string {
	keys := sortedKeys(params)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		if key == "sign" || key == "sign_type" || params[key] == "" {
			continue
		}
		parts = append(parts, key+"="+params[key])
	}
	return strings.Join(parts, "&")
}

func sortedKeys(params map[string]string) []string {
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func parseRSAPrivateKey(text string) (*rsa.PrivateKey, error) {
	block, err := pemBlock(text, "PRIVATE KEY")
	if err != nil {
		return nil, err
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if rsaKey, ok := key.(*rsa.PrivateKey); ok {
			return rsaKey, nil
		}
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func parseRSAPublicKey(text string) (*rsa.PublicKey, error) {
	block, err := pemBlock(text, "PUBLIC KEY")
	if err != nil {
		return nil, err
	}
	if key, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		if rsaKey, ok := key.(*rsa.PublicKey); ok {
			return rsaKey, nil
		}
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err == nil {
		if rsaKey, ok := cert.PublicKey.(*rsa.PublicKey); ok {
			return rsaKey, nil
		}
	}
	return nil, fmt.Errorf("invalid RSA public key")
}

func pemBlock(text string, blockType string) (*pem.Block, error) {
	cleaned := strings.ReplaceAll(strings.TrimSpace(text), `\n`, "\n")
	if !strings.Contains(cleaned, "-----BEGIN") {
		cleaned = "-----BEGIN " + blockType + "-----\n" + cleaned + "\n-----END " + blockType + "-----"
	}
	block, _ := pem.Decode([]byte(cleaned))
	if block == nil {
		return nil, fmt.Errorf("invalid PEM block")
	}
	return block, nil
}

func escapeJSON(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return value
}

func htmlEscape(value string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&#39;")
	return replacer.Replace(value)
}

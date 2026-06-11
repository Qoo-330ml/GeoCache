package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestVerifyLicenseReturnsSignedMemberLicense(t *testing.T) {
	a, publicKey := testLicenseApp(t)
	plainCode := "QMBY-TEST-0001"
	if err := a.db.Create(&ActivationCode{
		CodeHash:     hashCode(plainCode),
		CodePrefix:   codePrefix(plainCode),
		Email:        "user@example.com",
		Level:        LevelYearly,
		DurationDays: levelDurationDays[LevelYearly],
		Status:       StatusIssued,
	}).Error; err != nil {
		t.Fatalf("create activation code: %v", err)
	}

	res := postVerifyLicense(t, a, `{
		"activation_code": "`+plainCode+`",
		"instance_id": "qmby-test-instance",
		"beijing_time": "2026-06-04 15:30:00",
		"qmby_version": "0.0.28"
	}`)
	if res.Code != http.StatusOK {
		t.Fatalf("verify status = %d, body = %s", res.Code, res.Body.String())
	}

	var envelope signedLicenseResponse
	if err := json.Unmarshal(res.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if envelope.Signature == "" {
		t.Fatal("signature is empty")
	}
	if !ed25519.Verify(publicKey, envelope.License, mustDecodeSignature(t, envelope.Signature)) {
		t.Fatal("signature does not verify over license JSON bytes")
	}
	if ok := clientAcceptsSignedLicense(t, envelope, publicKey, "user@example.com", "qmby-test-instance", time.Now().In(beijingLocation())); !ok {
		t.Fatal("client rejected signed member license")
	}

	var top map[string]json.RawMessage
	if err := json.Unmarshal(res.Body.Bytes(), &top); err != nil {
		t.Fatalf("decode top-level response: %v", err)
	}
	if _, ok := top["member"]; ok {
		t.Fatal("top-level member must not be present")
	}
	if _, ok := top["features"]; ok {
		t.Fatal("top-level features must not be present")
	}
}

func TestSignedLicenseClientRejectsEmailMismatch(t *testing.T) {
	a, publicKey := testLicenseApp(t)
	plainCode := "QMBY-TEST-0002"
	if err := a.db.Create(&ActivationCode{
		CodeHash:     hashCode(plainCode),
		CodePrefix:   codePrefix(plainCode),
		Email:        "user@example.com",
		Level:        LevelPermanent,
		DurationDays: levelDurationDays[LevelPermanent],
		Status:       StatusIssued,
	}).Error; err != nil {
		t.Fatalf("create activation code: %v", err)
	}

	envelope := decodeVerifyEnvelope(t, postVerifyLicense(t, a, `{
		"activation_code": "`+plainCode+`",
		"instance_id": "qmby-test-instance"
	}`))
	if clientAcceptsSignedLicense(t, envelope, publicKey, "other@example.com", "qmby-test-instance", time.Now().In(beijingLocation())) {
		t.Fatal("client accepted license whose email does not match the request")
	}
}

func TestSignedLicenseClientRejectsInstanceIDMismatch(t *testing.T) {
	a, publicKey := testLicenseApp(t)
	plainCode := "QMBY-TEST-0003"
	if err := a.db.Create(&ActivationCode{
		CodeHash:     hashCode(plainCode),
		CodePrefix:   codePrefix(plainCode),
		Email:        "user@example.com",
		Level:        LevelPermanent,
		DurationDays: levelDurationDays[LevelPermanent],
		Status:       StatusIssued,
	}).Error; err != nil {
		t.Fatalf("create activation code: %v", err)
	}

	envelope := decodeVerifyEnvelope(t, postVerifyLicense(t, a, `{
		"activation_code": "`+plainCode+`",
		"instance_id": "qmby-test-instance"
	}`))
	if clientAcceptsSignedLicense(t, envelope, publicKey, "user@example.com", "other-instance", time.Now().In(beijingLocation())) {
		t.Fatal("client accepted license whose instance_id does not match the request")
	}
}

func TestVerifyLicenseExpiresPastMember(t *testing.T) {
	a, publicKey := testLicenseApp(t)
	loc := beijingLocation()
	startsAt := time.Now().In(loc).Add(-48 * time.Hour)
	expiresAt := time.Now().In(loc).Add(-24 * time.Hour)
	plainCode := "QMBY-TEST-0004"
	if err := a.db.Create(&ActivationCode{
		CodeHash:     hashCode(plainCode),
		CodePrefix:   codePrefix(plainCode),
		Email:        "expired@example.com",
		Level:        LevelYearly,
		DurationDays: levelDurationDays[LevelYearly],
		Status:       StatusActive,
		StartsAt:     &startsAt,
		ExpiresAt:    &expiresAt,
	}).Error; err != nil {
		t.Fatalf("create activation code: %v", err)
	}

	envelope := decodeVerifyEnvelope(t, postVerifyLicense(t, a, `{
		"activation_code": "`+plainCode+`",
		"instance_id": "qmby-expired-instance"
	}`))
	if !ed25519.Verify(publicKey, envelope.License, mustDecodeSignature(t, envelope.Signature)) {
		t.Fatal("signature does not verify over expired license JSON bytes")
	}
	var license licensePayload
	if err := json.Unmarshal(envelope.License, &license); err != nil {
		t.Fatalf("decode license: %v", err)
	}
	if license.Member && license.Status != StatusExpired {
		t.Fatalf("expired license returned member=true without expired status: %+v", license)
	}
	if clientAcceptsSignedLicense(t, envelope, publicKey, "expired@example.com", "qmby-expired-instance", time.Now().In(loc)) {
		t.Fatal("client accepted expired license as active member")
	}
}

func TestVerifyLicenseRejectsDifferentInstanceAfterActivation(t *testing.T) {
	a, publicKey := testLicenseApp(t)
	plainCode := "QMBY-TEST-0005"
	if err := a.db.Create(&ActivationCode{
		CodeHash:     hashCode(plainCode),
		CodePrefix:   codePrefix(plainCode),
		Email:        "user@example.com",
		Level:        LevelPermanent,
		DurationDays: levelDurationDays[LevelPermanent],
		Status:       StatusIssued,
	}).Error; err != nil {
		t.Fatalf("create activation code: %v", err)
	}

	first := decodeVerifyEnvelope(t, postVerifyLicense(t, a, `{
		"activation_code": "`+plainCode+`",
		"instance_id": "qmby-first-instance"
	}`))
	if !clientAcceptsSignedLicense(t, first, publicKey, "user@example.com", "qmby-first-instance", time.Now().In(beijingLocation())) {
		t.Fatal("first instance did not activate")
	}

	second := decodeVerifyEnvelope(t, postVerifyLicense(t, a, `{
		"activation_code": "`+plainCode+`",
		"instance_id": "qmby-second-instance"
	}`))
	var license licensePayload
	if err := json.Unmarshal(second.License, &license); err != nil {
		t.Fatalf("decode license: %v", err)
	}
	if license.Member {
		t.Fatalf("second instance reused activation code: %+v", license)
	}
	if license.Status != "inactive" {
		t.Fatalf("status = %q, want inactive", license.Status)
	}
}

func testLicenseApp(t *testing.T) (*app, ed25519.PublicKey) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := openDatabase(filepath.Join(t.TempDir(), "license.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return &app{db: db, licenseAPIKey: "test-key", signingKey: privateKey}, publicKey
}

func postVerifyLicense(t *testing.T, a *app, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := gin.New()
	r.POST("/api/license/verify", a.requireLicenseKey(), a.verifyLicense)
	req := httptest.NewRequest(http.MethodPost, "/api/license/verify", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-key")
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	return res
}

func decodeVerifyEnvelope(t *testing.T, res *httptest.ResponseRecorder) signedLicenseResponse {
	t.Helper()
	if res.Code != http.StatusOK {
		t.Fatalf("verify status = %d, body = %s", res.Code, res.Body.String())
	}
	var envelope signedLicenseResponse
	if err := json.Unmarshal(res.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return envelope
}

func clientAcceptsSignedLicense(t *testing.T, envelope signedLicenseResponse, publicKey ed25519.PublicKey, email, instanceID string, now time.Time) bool {
	t.Helper()
	signature, err := base64.StdEncoding.DecodeString(envelope.Signature)
	if err != nil {
		return false
	}
	if !ed25519.Verify(publicKey, envelope.License, signature) {
		return false
	}
	var license licensePayload
	if err := json.Unmarshal(envelope.License, &license); err != nil {
		return false
	}
	if license.Email != normalizeEmail(email) {
		return false
	}
	if license.InstanceID != strings.TrimSpace(instanceID) {
		return false
	}
	if license.StartsAt != nil && license.StartsAt.After(now) {
		return false
	}
	if license.ExpiresAt != nil && !license.ExpiresAt.After(now) {
		return false
	}
	return license.Member
}

func mustDecodeSignature(t *testing.T, value string) []byte {
	t.Helper()
	signature, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		t.Fatalf("decode signature: %v", err)
	}
	return signature
}

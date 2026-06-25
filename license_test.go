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
	var license licensePayload
	if err := json.Unmarshal(envelope.License, &license); err != nil {
		t.Fatalf("decode license: %v", err)
	}
	if license.QmbyVersion != "0.0.28" {
		t.Fatalf("qmby_version = %q, want 0.0.28", license.QmbyVersion)
	}
	if license.ValidUntil.Sub(license.IssuedAt) > 24*time.Hour || !license.ValidUntil.After(license.IssuedAt) {
		t.Fatalf("unexpected rolling validity window: issued=%s valid_until=%s", license.IssuedAt, license.ValidUntil)
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

func TestMigrateLegacyCodesToBeta(t *testing.T) {
	db, err := openDatabase(filepath.Join(t.TempDir(), "license.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	expiresAt := time.Now().Add(-time.Hour)
	codes := []ActivationCode{
		{CodeHash: hashCode("QMBY-LEGACY-0001"), CodePrefix: "QMBY-LEG", Email: "yearly@example.com", Level: LevelYearly, DurationDays: 365, Status: StatusExpired, ExpiresAt: &expiresAt},
		{CodeHash: hashCode("QMBY-LEGACY-0002"), CodePrefix: "QMBY-LEG", Email: "permanent@example.com", Level: LevelPermanent, DurationDays: 0, Status: StatusActive},
		{CodeHash: hashCode("QMBY-LEGACY-0003"), CodePrefix: "QMBY-LEG", Email: "disabled@example.com", Level: LevelTrial, DurationDays: 7, Status: StatusDisabled, ExpiresAt: &expiresAt},
	}
	if err := db.Create(&codes).Error; err != nil {
		t.Fatalf("create legacy codes: %v", err)
	}
	if err := migrateLegacyCodesToBeta(db); err != nil {
		t.Fatalf("migrate legacy codes: %v", err)
	}
	var migrated []ActivationCode
	if err := db.Order("email ASC").Find(&migrated).Error; err != nil {
		t.Fatalf("load migrated codes: %v", err)
	}
	for _, code := range migrated {
		if code.Level != LevelBeta || code.DurationDays != 0 || code.ExpiresAt != nil {
			t.Fatalf("code was not migrated to permanent beta: %+v", code)
		}
		if code.Email == "disabled@example.com" && code.Status != StatusDisabled {
			t.Fatalf("disabled code status = %q, want %q", code.Status, StatusDisabled)
		}
		if code.Email != "disabled@example.com" && code.Status != StatusActive {
			t.Fatalf("legacy code status = %q, want %q", code.Status, StatusActive)
		}
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
		"instance_id": "qmby-test-instance",
		"qmby_version": "0.0.39"
	}`))
	if clientAcceptsSignedLicense(t, envelope, publicKey, "other@example.com", "qmby-test-instance", time.Now().In(beijingLocation())) {
		t.Fatal("client accepted license whose email does not match the request")
	}
}

func TestRequestTrialCodeCreatesSevenDayPlusCode(t *testing.T) {
	a, _ := testLicenseApp(t)
	res := postTrialCode(t, a, `{"email":"Trial@Example.com","instance_id":"qmby-trial-instance","qmby_version":"0.0.39"}`)
	if res.Code != http.StatusOK {
		t.Fatalf("trial status = %d, body = %s", res.Code, res.Body.String())
	}
	var data struct {
		ActivationCode string `json:"activation_code"`
		Level          string `json:"level"`
		DurationDays   int    `json:"duration_days"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &data); err != nil {
		t.Fatalf("decode trial response: %v", err)
	}
	if data.ActivationCode == "" {
		t.Fatal("activation code is empty")
	}
	if data.Level != LevelPlus || data.DurationDays != 7 {
		t.Fatalf("trial code = level %q duration %d, want plus 7", data.Level, data.DurationDays)
	}
	var code ActivationCode
	if err := a.db.Where("code_hash = ?", hashCode(data.ActivationCode)).First(&code).Error; err != nil {
		t.Fatalf("load trial code: %v", err)
	}
	if code.Email != "trial@example.com" || code.Status != StatusIssued || code.Note != "trial_instance:qmby-trial-instance" {
		t.Fatalf("unexpected stored trial code: %+v", code)
	}
}

func TestRequestTrialCodeRejectsRepeatedInstance(t *testing.T) {
	a, _ := testLicenseApp(t)
	first := postTrialCode(t, a, `{"email":"trial@example.com","instance_id":"qmby-trial-instance"}`)
	if first.Code != http.StatusOK {
		t.Fatalf("first trial status = %d, body = %s", first.Code, first.Body.String())
	}
	second := postTrialCode(t, a, `{"email":"other@example.com","instance_id":"qmby-trial-instance"}`)
	if second.Code != http.StatusConflict {
		t.Fatalf("second trial status = %d, body = %s", second.Code, second.Body.String())
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
		"instance_id": "qmby-test-instance",
		"qmby_version": "0.0.39"
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
		"instance_id": "qmby-expired-instance",
		"qmby_version": "0.0.39"
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

func TestVerifyLicenseAllowsLevelInstanceLimit(t *testing.T) {
	tests := []struct {
		name      string
		plainCode string
		level     string
		limit     int
	}{
		{name: "plus", plainCode: "QMBY-TEST-0005", level: LevelPlus, limit: 1},
		{name: "pro", plainCode: "QMBY-TEST-0006", level: LevelPro, limit: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, publicKey := testLicenseApp(t)
			if err := a.db.Create(&ActivationCode{
				CodeHash:     hashCode(tt.plainCode),
				CodePrefix:   codePrefix(tt.plainCode),
				Email:        "user@example.com",
				Level:        tt.level,
				DurationDays: levelDurationDays[tt.level],
				Status:       StatusIssued,
			}).Error; err != nil {
				t.Fatalf("create activation code: %v", err)
			}

			for i := 1; i <= tt.limit; i++ {
				instanceID := "qmby-instance-" + string(rune('0'+i))
				envelope := decodeVerifyEnvelope(t, postVerifyLicense(t, a, `{
					"activation_code": "`+tt.plainCode+`",
					"instance_id": "`+instanceID+`",
					"qmby_version": "0.0.39"
				}`))
				if !clientAcceptsSignedLicense(t, envelope, publicKey, "user@example.com", instanceID, time.Now().In(beijingLocation())) {
					t.Fatalf("instance %d did not activate", i)
				}
			}

			blockedInstanceID := "qmby-instance-blocked"
			blocked := decodeVerifyEnvelope(t, postVerifyLicense(t, a, `{
				"activation_code": "`+tt.plainCode+`",
				"instance_id": "`+blockedInstanceID+`",
				"qmby_version": "0.0.39"
			}`))
			var license licensePayload
			if err := json.Unmarshal(blocked.License, &license); err != nil {
				t.Fatalf("decode license: %v", err)
			}
			if license.Member {
				t.Fatalf("instance over %s limit activated: %+v", tt.name, license)
			}
			if license.Status != "inactive" {
				t.Fatalf("status = %q, want inactive", license.Status)
			}
		})
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

func postTrialCode(t *testing.T, a *app, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := gin.New()
	r.POST("/api/license/trial", a.requireLicenseKey(), a.requestTrialCode)
	req := httptest.NewRequest(http.MethodPost, "/api/license/trial", strings.NewReader(body))
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
	if strings.TrimSpace(license.QmbyVersion) == "" {
		return false
	}
	if now.Before(license.IssuedAt.Add(-2*time.Minute)) || !now.Before(license.ValidUntil) {
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

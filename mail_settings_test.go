package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/smtp"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCreateCodeSendsActivationCodeWithSavedMailSettings(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := openDatabase(filepath.Join(t.TempDir(), "license.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	var sentAddr, sentFrom string
	var sentTo []string
	var sentMessage string
	a := &app{
		db:            db,
		adminUser:     "admin",
		adminPassword: "secret",
		mailer: smtpMailer{sendMail: func(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
			sentAddr = addr
			sentFrom = from
			sentTo = to
			sentMessage = string(msg)
			return nil
		}},
	}
	if err := db.Save(&MailSetting{
		ID:       mailSettingID,
		Host:     "smtp.example.com",
		Port:     "2525",
		Username: "mailer@example.com",
		Password: "smtp-secret",
		From:     "Qmby License <mailer@example.com>",
	}).Error; err != nil {
		t.Fatalf("save mail setting: %v", err)
	}

	r := gin.New()
	r.POST("/api/admin/codes", a.basicAuth(), a.createCode)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/codes", strings.NewReader(`{"email":"User@Example.com","level":"trial"}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("admin", "secret")
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("create code status = %d, body = %s", res.Code, res.Body.String())
	}

	var body struct {
		PlainCode string `json:"plain_code"`
		MailSent  bool   `json:"mail_sent"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !body.MailSent {
		t.Fatalf("mail_sent = false, body = %s", res.Body.String())
	}
	if sentAddr != "smtp.example.com:2525" {
		t.Fatalf("sent addr = %q", sentAddr)
	}
	if sentFrom != "mailer@example.com" {
		t.Fatalf("sent from = %q", sentFrom)
	}
	if len(sentTo) != 1 || sentTo[0] != "user@example.com" {
		t.Fatalf("sent to = %#v", sentTo)
	}
	if body.PlainCode == "" || !strings.Contains(sentMessage, body.PlainCode) {
		t.Fatalf("sent message does not contain activation code: %q", sentMessage)
	}
}

func TestCreateCodeAcceptsAllowedDurations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := openDatabase(filepath.Join(t.TempDir(), "license.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	a := &app{db: db, adminUser: "admin", adminPassword: "secret"}
	r := gin.New()
	r.POST("/api/admin/codes", a.basicAuth(), a.createCode)

	for _, tc := range []struct {
		name         string
		durationDays int
	}{
		{name: "seven_days", durationDays: 7},
		{name: "one_month", durationDays: 30},
		{name: "one_year", durationDays: 365},
		{name: "permanent", durationDays: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"email":"` + tc.name + `@example.com","level":"plus","duration_days":` + strconv.Itoa(tc.durationDays) + `}`
			req := httptest.NewRequest(http.MethodPost, "/api/admin/codes", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.SetBasicAuth("admin", "secret")
			res := httptest.NewRecorder()
			r.ServeHTTP(res, req)
			if res.Code != http.StatusOK {
				t.Fatalf("create code status = %d, body = %s", res.Code, res.Body.String())
			}

			var code ActivationCode
			if err := db.Where("email = ?", tc.name+"@example.com").First(&code).Error; err != nil {
				t.Fatalf("load activation code: %v", err)
			}
			if code.DurationDays != tc.durationDays {
				t.Fatalf("duration_days = %d, want %d", code.DurationDays, tc.durationDays)
			}
		})
	}
}

func TestCreateCodeRejectsInvalidDuration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := openDatabase(filepath.Join(t.TempDir(), "license.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	a := &app{db: db, adminUser: "admin", adminPassword: "secret"}
	r := gin.New()
	r.POST("/api/admin/codes", a.basicAuth(), a.createCode)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/codes", strings.NewReader(`{"email":"invalid@example.com","level":"plus","duration_days":31}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("admin", "secret")
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("create code status = %d, want %d, body = %s", res.Code, http.StatusBadRequest, res.Body.String())
	}
}

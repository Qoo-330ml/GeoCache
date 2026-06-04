package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSubmitAndExportOrganizerFailedRecords(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := openDatabase(filepath.Join(t.TempDir(), "license.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	a := &app{db: db, licenseAPIKey: "test-key"}

	r := gin.New()
	r.POST("/api/organizer/failed-records", a.requireLicenseKey(), a.submitOrganizerFailedRecords)
	r.GET("/api/admin/organizer/failed-records/export", a.exportOrganizerFailedRecords)

	body := `{
		"email": "User@Example.com",
		"beijing_time": "2026-06-04 15:30:00",
		"instance_id": "qmby-0123456789abcdef0123456789abcdef",
		"qmby_version": "0.0.28-fix1",
		"records": "2026-06-04T15:29:10+08:00\tfailed\tMovie.mkv\t/video/source/Movie.mkv\tTMDB search failed\nmalformed line"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/organizer/failed-records", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-key")
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("submit status = %d, body = %s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), `"count":2`) {
		t.Fatalf("submit response missing count: %s", res.Body.String())
	}

	var total int64
	if err := db.Model(&OrganizerFailedRecord{}).Count(&total).Error; err != nil {
		t.Fatalf("count records: %v", err)
	}
	if total != 2 {
		t.Fatalf("record count = %d, want 2", total)
	}

	var malformed OrganizerFailedRecord
	if err := db.Where("line_number = ?", 2).First(&malformed).Error; err != nil {
		t.Fatalf("load malformed row: %v", err)
	}
	if malformed.ParseError == "" {
		t.Fatal("malformed row parse_error is empty")
	}

	exportReq := httptest.NewRequest(http.MethodGet, "/api/admin/organizer/failed-records/export", nil)
	exportRes := httptest.NewRecorder()
	r.ServeHTTP(exportRes, exportReq)
	if exportRes.Code != http.StatusOK {
		t.Fatalf("export status = %d, body = %s", exportRes.Code, exportRes.Body.String())
	}
	csvText := exportRes.Body.String()
	if !strings.Contains(csvText, "Movie.mkv") || !strings.Contains(csvText, "malformed line") {
		t.Fatalf("export missing records: %s", csvText)
	}
}

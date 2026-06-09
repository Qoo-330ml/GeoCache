package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestQshareRequiresContributionBeforeBrowsing(t *testing.T) {
	a := testQshareApp(t)
	createActiveQshareLicense(t, a, "user@example.com")
	r := testQshareRouter(a)

	res := qshareRequest(t, r, http.MethodGet, "/api/qshare/resources?email=user@example.com&instance_id=qmby-a", "")
	if res.Code != http.StatusForbidden {
		t.Fatalf("list status = %d, body = %s", res.Code, res.Body.String())
	}
}

func TestQsharePublishListDetailAndAnonymousSource(t *testing.T) {
	a := testQshareApp(t)
	createActiveQshareLicense(t, a, "user@example.com")
	r := testQshareRouter(a)

	publish := qshareRequest(t, r, http.MethodPost, "/api/qshare/resources", sampleQshareBody("user@example.com", "qmby-a", "Interstellar"))
	if publish.Code != http.StatusOK {
		t.Fatalf("publish status = %d, body = %s", publish.Code, publish.Body.String())
	}
	var published struct {
		Resource qshareResourceDetail `json:"resource"`
	}
	if err := json.Unmarshal(publish.Body.Bytes(), &published); err != nil {
		t.Fatalf("decode publish: %v", err)
	}
	if published.Resource.SourceID == "" || strings.Contains(publish.Body.String(), "user@example.com") || strings.Contains(publish.Body.String(), "qmby-a") {
		t.Fatalf("publish response leaks publisher identity: %s", publish.Body.String())
	}

	list := qshareRequest(t, r, http.MethodGet, "/api/qshare/resources?email=user@example.com&instance_id=qmby-a", "")
	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", list.Code, list.Body.String())
	}
	if !strings.Contains(list.Body.String(), `"file_count":1`) || strings.Contains(list.Body.String(), "user@example.com") || strings.Contains(list.Body.String(), "qmby-a") {
		t.Fatalf("list response invalid or leaks identity: %s", list.Body.String())
	}

	detail := qshareRequest(t, r, http.MethodGet, "/api/qshare/resources/"+strconvID(published.Resource.ID)+"?email=user@example.com&instance_id=qmby-a", "")
	if detail.Code != http.StatusOK {
		t.Fatalf("detail status = %d, body = %s", detail.Code, detail.Body.String())
	}
	if !strings.Contains(detail.Body.String(), `"sha1":"`) || strings.Contains(detail.Body.String(), "user@example.com") || strings.Contains(detail.Body.String(), "qmby-a") {
		t.Fatalf("detail response invalid or leaks identity: %s", detail.Body.String())
	}
}

func TestQshareOnlyOwnerCanUpdateOrCancel(t *testing.T) {
	a := testQshareApp(t)
	createActiveQshareLicense(t, a, "owner@example.com")
	createActiveQshareLicense(t, a, "other@example.com")
	r := testQshareRouter(a)

	publish := qshareRequest(t, r, http.MethodPost, "/api/qshare/resources", sampleQshareBody("owner@example.com", "qmby-owner", "Original"))
	if publish.Code != http.StatusOK {
		t.Fatalf("publish status = %d, body = %s", publish.Code, publish.Body.String())
	}
	var published struct {
		Resource qshareResourceDetail `json:"resource"`
	}
	if err := json.Unmarshal(publish.Body.Bytes(), &published); err != nil {
		t.Fatalf("decode publish: %v", err)
	}

	update := qshareRequest(t, r, http.MethodPut, "/api/qshare/resources/"+strconvID(published.Resource.ID), sampleQshareBody("other@example.com", "qmby-other", "Hijack"))
	if update.Code != http.StatusNotFound {
		t.Fatalf("other update status = %d, body = %s", update.Code, update.Body.String())
	}

	cancel := qshareRequest(t, r, http.MethodDelete, "/api/qshare/resources/"+strconvID(published.Resource.ID)+"?email=other@example.com&instance_id=qmby-other", "")
	if cancel.Code != http.StatusNotFound {
		t.Fatalf("other cancel status = %d, body = %s", cancel.Code, cancel.Body.String())
	}

	cancel = qshareRequest(t, r, http.MethodDelete, "/api/qshare/resources/"+strconvID(published.Resource.ID)+"?email=owner@example.com&instance_id=qmby-owner", "")
	if cancel.Code != http.StatusOK {
		t.Fatalf("owner cancel status = %d, body = %s", cancel.Code, cancel.Body.String())
	}
}

func testQshareApp(t *testing.T) *app {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := openDatabase(filepath.Join(t.TempDir(), "license.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	return &app{db: db, licenseAPIKey: "test-key"}
}

func testQshareRouter(a *app) *gin.Engine {
	r := gin.New()
	r.GET("/api/qshare/resources", a.requireLicenseKey(), a.listQshareResources)
	r.GET("/api/qshare/resources/:id", a.requireLicenseKey(), a.getQshareResource)
	r.POST("/api/qshare/resources", a.requireLicenseKey(), a.publishQshareResource)
	r.PUT("/api/qshare/resources/:id", a.requireLicenseKey(), a.updateQshareResource)
	r.DELETE("/api/qshare/resources/:id", a.requireLicenseKey(), a.cancelQshareResource)
	return r
}

func createActiveQshareLicense(t *testing.T, a *app, email string) {
	t.Helper()
	now := time.Now().In(beijingLocation()).Add(-time.Hour)
	if err := a.db.Create(&ActivationCode{
		CodeHash:     hashCode("QSHARE-TEST-" + normalizeEmail(email)),
		CodePrefix:   "QSHARE-TEST",
		Email:        normalizeEmail(email),
		Level:        LevelYearly,
		DurationDays: levelDurationDays[LevelYearly],
		Status:       StatusActive,
		StartsAt:     &now,
		ExpiresAt:    ptrTime(now.AddDate(0, 0, 30)),
	}).Error; err != nil {
		t.Fatalf("create activation code: %v", err)
	}
}

func qshareRequest(t *testing.T, r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer test-key")
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	return res
}

func sampleQshareBody(email, instanceID, title string) string {
	return `{
		"email": "` + email + `",
		"instance_id": "` + instanceID + `",
		"title": "` + title + `",
		"media_type": "movie",
		"tmdb_id": "157336",
		"year": 2014,
		"poster_url": "https://image.tmdb.org/t/p/w500/poster.jpg",
		"files": [{
			"name": "Interstellar.mkv",
			"size": 123456789,
			"sha1": "0123456789abcdef0123456789abcdef01234567",
			"relative_path": "Interstellar/Interstellar.mkv"
		}]
	}`
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

func strconvID(id uint) string {
	return strconv.FormatUint(uint64(id), 10)
}

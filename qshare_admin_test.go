package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAdminQshareListEditAndUnpublish(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := openDatabase(filepath.Join(t.TempDir(), "license.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	a := &app{db: db, adminUser: "admin", adminPassword: "secret"}
	resource := QshareResource{
		PublisherEmail: "owner@example.com",
		InstanceID:     "qmby-owner",
		MediaType:      "movie",
		TMDBID:         "157336",
		Title:          "Original",
		Year:           2014,
		PosterURL:      "https://example.com/original.jpg",
		SourcePath:     "Movies/Original",
		FileCount:      1,
		TotalSize:      1024,
		Status:         qshareStatusPublished,
	}
	if err := db.Create(&resource).Error; err != nil {
		t.Fatalf("create resource: %v", err)
	}
	if err := db.Create(&QshareFile{ResourceID: resource.ID, Name: "movie.mkv", RelativePath: "movie.mkv", Size: 1024, SHA1: "ABCDEF"}).Error; err != nil {
		t.Fatalf("create file: %v", err)
	}

	router := gin.New()
	admin := router.Group("/api/admin")
	admin.Use(a.basicAuth())
	admin.GET("/qshare/resources", a.listAdminQshareResources)
	admin.GET("/qshare/resources/:id", a.getAdminQshareResource)
	admin.PUT("/qshare/resources/:id", a.updateAdminQshareResource)
	admin.POST("/qshare/resources/:id/unpublish", a.unpublishAdminQshareResource)

	list := adminQshareRequest(t, router, http.MethodGet, "/api/admin/qshare/resources", "")
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), `"publisher_email":"owner@example.com"`) {
		t.Fatalf("admin list invalid: code = %d, body = %s", list.Code, list.Body.String())
	}

	detail := adminQshareRequest(t, router, http.MethodGet, "/api/admin/qshare/resources/"+strconv.FormatUint(uint64(resource.ID), 10), "")
	if detail.Code != http.StatusOK || !strings.Contains(detail.Body.String(), `"files":[`) {
		t.Fatalf("admin detail invalid: code = %d, body = %s", detail.Code, detail.Body.String())
	}

	update := adminQshareRequest(t, router, http.MethodPut, "/api/admin/qshare/resources/"+strconv.FormatUint(uint64(resource.ID), 10), `{
		"media_type":"movie",
		"tmdb_id":"157336",
		"title":"Admin Updated",
		"year":2014,
		"poster_url":"https://example.com/poster.jpg",
		"source_path":"Movies/Admin Updated"
	}`)
	if update.Code != http.StatusOK || !strings.Contains(update.Body.String(), `"title":"Admin Updated"`) {
		t.Fatalf("admin update invalid: code = %d, body = %s", update.Code, update.Body.String())
	}

	unpublish := adminQshareRequest(t, router, http.MethodPost, "/api/admin/qshare/resources/"+strconv.FormatUint(uint64(resource.ID), 10)+"/unpublish", "")
	if unpublish.Code != http.StatusOK {
		t.Fatalf("admin unpublish invalid: code = %d, body = %s", unpublish.Code, unpublish.Body.String())
	}
	if err := a.db.First(&resource, resource.ID).Error; err != nil {
		t.Fatalf("load unpublished resource: %v", err)
	}
	if resource.Status != qshareStatusDeleted {
		t.Fatalf("resource status = %q, want %q", resource.Status, qshareStatusDeleted)
	}
}

func adminQshareRequest(t *testing.T, router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.SetBasicAuth("admin", "secret")
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	return res
}

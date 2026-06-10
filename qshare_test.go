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

func TestQshareStatusAndListUsePublishFolderConfigured(t *testing.T) {
	a := testQshareApp(t)
	createActiveQshareLicense(t, a, "user@example.com")
	r := testQshareRouter(a)

	status := qshareRequest(t, r, "/api/qshare/status", qshareBaseBody("user@example.com", "qmby-a", true))
	if status.Code != http.StatusOK {
		t.Fatalf("status code = %d, body = %s", status.Code, status.Body.String())
	}
	if !strings.Contains(status.Body.String(), `"enabled":true`) || !strings.Contains(status.Body.String(), `"can_browse":true`) {
		t.Fatalf("unexpected status response: %s", status.Body.String())
	}

	list := qshareRequest(t, r, "/api/qshare/resources/list", qshareBaseBody("user@example.com", "qmby-a", true))
	if list.Code != http.StatusOK {
		t.Fatalf("list code = %d, body = %s", list.Code, list.Body.String())
	}
	if !strings.Contains(list.Body.String(), `"can_browse":true`) || !strings.Contains(list.Body.String(), `"resources":[]`) {
		t.Fatalf("configured folder should allow browsing empty qshare: %s", list.Body.String())
	}

	disabled := qshareRequest(t, r, "/api/qshare/resources/list", qshareBaseBody("user@example.com", "qmby-a", false))
	if disabled.Code != http.StatusOK {
		t.Fatalf("disabled list code = %d, body = %s", disabled.Code, disabled.Body.String())
	}
	if !strings.Contains(disabled.Body.String(), `"can_browse":true`) || !strings.Contains(disabled.Body.String(), `"resources":[]`) {
		t.Fatalf("list should remain available without publish folder: %s", disabled.Body.String())
	}
}

func TestQsharePublishUpdatesDuplicateAndHidesOwnerIdentity(t *testing.T) {
	a := testQshareApp(t)
	createActiveQshareLicense(t, a, "user@example.com")
	r := testQshareRouter(a)

	first := qshareRequest(t, r, "/api/qshare/resources/publish", sampleQsharePublishBody("user@example.com", "qmby-a", 0, "Original"))
	if first.Code != http.StatusOK {
		t.Fatalf("publish code = %d, body = %s", first.Code, first.Body.String())
	}
	var firstBody struct {
		Resource qshareResourceResponse `json:"resource"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &firstBody); err != nil {
		t.Fatalf("decode first publish: %v", err)
	}
	if firstBody.Resource.ID == 0 {
		t.Fatal("published resource id is empty")
	}

	second := qshareRequest(t, r, "/api/qshare/resources/publish", sampleQsharePublishBodyWithSource("user@example.com", "qmby-a", 0, "Updated", "115://Movies/RenamedInterstellar"))
	if second.Code != http.StatusOK {
		t.Fatalf("republish code = %d, body = %s", second.Code, second.Body.String())
	}
	var secondBody struct {
		Resource qshareResourceResponse `json:"resource"`
	}
	if err := json.Unmarshal(second.Body.Bytes(), &secondBody); err != nil {
		t.Fatalf("decode second publish: %v", err)
	}
	if secondBody.Resource.ID != firstBody.Resource.ID || secondBody.Resource.Title != "Updated" {
		t.Fatalf("duplicate publish did not update original: %+v", secondBody.Resource)
	}
	if strings.Contains(second.Body.String(), "user@example.com") || strings.Contains(second.Body.String(), "qmby-a") {
		t.Fatalf("publish response leaks owner identity: %s", second.Body.String())
	}
}

func TestQshareListDetailAndDelete(t *testing.T) {
	a := testQshareApp(t)
	createActiveQshareLicense(t, a, "owner@example.com")
	createActiveQshareLicense(t, a, "other@example.com")
	r := testQshareRouter(a)

	publish := qshareRequest(t, r, "/api/qshare/resources/publish", sampleQsharePublishBody("owner@example.com", "qmby-owner", 0, "Interstellar"))
	if publish.Code != http.StatusOK {
		t.Fatalf("publish code = %d, body = %s", publish.Code, publish.Body.String())
	}
	var published struct {
		Resource qshareResourceResponse `json:"resource"`
	}
	if err := json.Unmarshal(publish.Body.Bytes(), &published); err != nil {
		t.Fatalf("decode publish: %v", err)
	}

	list := qshareRequest(t, r, "/api/qshare/resources/list", qshareBaseBody("owner@example.com", "qmby-owner", true))
	if list.Code != http.StatusOK {
		t.Fatalf("list code = %d, body = %s", list.Code, list.Body.String())
	}
	if !strings.Contains(list.Body.String(), `"can_browse":true`) || strings.Contains(list.Body.String(), `"files"`) {
		t.Fatalf("list should be poster-wall only: %s", list.Body.String())
	}
	if strings.Contains(list.Body.String(), "owner@example.com") || strings.Contains(list.Body.String(), "qmby-owner") {
		t.Fatalf("list leaks owner identity: %s", list.Body.String())
	}

	detailWithoutPublishFolder := qshareRequest(t, r, "/api/qshare/resources/detail", qshareResourceIDBody("owner@example.com", "qmby-owner", false, published.Resource.ID))
	if detailWithoutPublishFolder.Code != http.StatusOK {
		t.Fatalf("detail without publish folder code = %d, body = %s", detailWithoutPublishFolder.Code, detailWithoutPublishFolder.Body.String())
	}

	detail := qshareRequest(t, r, "/api/qshare/resources/detail", qshareResourceIDBody("owner@example.com", "qmby-owner", true, published.Resource.ID))
	if detail.Code != http.StatusOK {
		t.Fatalf("detail code = %d, body = %s", detail.Code, detail.Body.String())
	}
	if !strings.Contains(detail.Body.String(), `"files":[`) || !strings.Contains(detail.Body.String(), `"sha1":"0123456789ABCDEF0123456789ABCDEF01234567"`) {
		t.Fatalf("detail missing files: %s", detail.Body.String())
	}
	if strings.Contains(detail.Body.String(), "chat_mid") || strings.Contains(detail.Body.String(), "chat_contact_id") {
		t.Fatalf("detail leaks hidden chat metadata: %s", detail.Body.String())
	}

	otherDelete := qshareRequest(t, r, "/api/qshare/resources/delete", qshareResourceIDBody("other@example.com", "qmby-other", true, published.Resource.ID))
	if otherDelete.Code != http.StatusNotFound {
		t.Fatalf("other delete code = %d, body = %s", otherDelete.Code, otherDelete.Body.String())
	}

	ownerDelete := qshareRequest(t, r, "/api/qshare/resources/delete", qshareResourceIDBody("owner@example.com", "qmby-owner", true, published.Resource.ID))
	if ownerDelete.Code != http.StatusOK || !strings.Contains(ownerDelete.Body.String(), `"success":true`) {
		t.Fatalf("owner delete failed: code = %d, body = %s", ownerDelete.Code, ownerDelete.Body.String())
	}
}

func TestQshareAcceptsStringIDs(t *testing.T) {
	a := testQshareApp(t)
	createActiveQshareLicense(t, a, "owner@example.com")
	r := testQshareRouter(a)

	body := strings.Replace(sampleQsharePublishBody("owner@example.com", "qmby-owner", 0, "Interstellar"), `"media_type"`, `"id":"","media_type"`, 1)
	publish := qshareRequest(t, r, "/api/qshare/resources/publish", body)
	if publish.Code != http.StatusOK {
		t.Fatalf("publish with string id = %d %s", publish.Code, publish.Body.String())
	}
	var published struct {
		Resource qshareResourceResponse `json:"resource"`
	}
	if err := json.Unmarshal(publish.Body.Bytes(), &published); err != nil {
		t.Fatalf("decode publish: %v", err)
	}

	detailBody := `{"email":"owner@example.com","instance_id":"qmby-owner","beijing_time":"2026-06-09 17:30:00","publish_folder_configured":true,"resource_id":"` + strconvUint(published.Resource.ID) + `"}`
	detail := qshareRequest(t, r, "/api/qshare/resources/detail", detailBody)
	if detail.Code != http.StatusOK {
		t.Fatalf("detail with string id = %d %s", detail.Code, detail.Body.String())
	}
}

func TestQshareAcceptsFileLevelPublisher(t *testing.T) {
	a := testQshareApp(t)
	createActiveQshareLicense(t, a, "owner@example.com")
	r := testQshareRouter(a)

	body := strings.Replace(sampleQsharePublishBody("owner@example.com", "qmby-owner", 0, "Interstellar"), `"publisher_115_id": "4577361",`, "", 1)
	body = strings.Replace(body, `"chat_mid": "mid-1"`, `"publisher_115_id": "4577361","chat_mid": "mid-1"`, 1)
	publish := qshareRequest(t, r, "/api/qshare/resources/publish", body)
	if publish.Code != http.StatusOK {
		t.Fatalf("publish code = %d, body = %s", publish.Code, publish.Body.String())
	}
	var published struct {
		Resource qshareResourceResponse `json:"resource"`
	}
	if err := json.Unmarshal(publish.Body.Bytes(), &published); err != nil {
		t.Fatalf("decode publish: %v", err)
	}
	if got := published.Resource.Files[0].Publisher115ID; got != "4577361" {
		t.Fatalf("file publisher_115_id = %q, want 4577361", got)
	}
}

func TestQshareAcceptsFolderFileItem(t *testing.T) {
	a := testQshareApp(t)
	createActiveQshareLicense(t, a, "owner@example.com")
	r := testQshareRouter(a)

	body := `{
		"email": "owner@example.com",
		"instance_id": "qmby-owner",
		"beijing_time": "2026-06-09 17:30:00",
		"publish_folder_configured": true,
		"resource": {
			"media_type": "tv",
			"tmdb_id": 123,
			"title": "灵魂摆渡·十年",
			"year": 2026,
			"poster_url": "https://image.tmdb.org/t/p/w500/poster.jpg",
			"publisher_115_id": "4577361",
			"file_count": 1,
			"total_size": 123456789,
			"files": [{
				"id": "DIR:Season 1",
				"name": "Season 1",
				"relative_path": "Season 1",
				"quality": "S01 24集 · 2160p WEB-DL HDR10+ HEVC",
				"is_dir": true,
				"size": 123456789,
				"sha1": "",
				"publisher_115_id": "4577361",
				"season_number": 1,
				"chat_mid": "mid-season-1",
				"chat_contact_id": "1182480"
			}]
		}
	}`
	publish := qshareRequest(t, r, "/api/qshare/resources/publish", body)
	if publish.Code != http.StatusOK {
		t.Fatalf("publish code = %d, body = %s", publish.Code, publish.Body.String())
	}
	var published struct {
		Resource qshareResourceResponse `json:"resource"`
	}
	if err := json.Unmarshal(publish.Body.Bytes(), &published); err != nil {
		t.Fatalf("decode publish: %v", err)
	}
	if len(published.Resource.Files) != 1 || !published.Resource.Files[0].IsDir {
		t.Fatalf("folder file item missing: %+v", published.Resource.Files)
	}
	if published.Resource.Files[0].Quality != "S01 24集 · 2160p WEB-DL HDR10+ HEVC" {
		t.Fatalf("quality = %q", published.Resource.Files[0].Quality)
	}
	if published.Resource.SourcePath != "" {
		t.Fatalf("source path should not be returned: %q", published.Resource.SourcePath)
	}
}

func TestMigrateQshareFilePublishersCopiesResourcePublisher(t *testing.T) {
	a := testQshareApp(t)
	resource := QshareResource{
		PublisherEmail: "owner@example.com", InstanceID: "qmby-owner", Publisher115ID: "4577361",
		SourceID: "source", Title: "Interstellar", MediaType: "movie", TMDBID: "157336",
		Year: 2014, PosterURL: "poster", SourcePath: "path", FileCount: 1, TotalSize: 1,
		Status: qshareStatusPublished,
	}
	if err := a.db.Create(&resource).Error; err != nil {
		t.Fatalf("create resource: %v", err)
	}
	file := QshareFile{ResourceID: resource.ID, Name: "Interstellar.mkv", Size: 1, SHA1: "ABC", RelativePath: "Interstellar.mkv"}
	if err := a.db.Create(&file).Error; err != nil {
		t.Fatalf("create file: %v", err)
	}
	if err := migrateQshareFilePublishers(a.db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	var migrated QshareFile
	if err := a.db.First(&migrated, file.ID).Error; err != nil {
		t.Fatalf("load migrated file: %v", err)
	}
	if migrated.Publisher115ID != "4577361" {
		t.Fatalf("migrated publisher_115_id = %q, want 4577361", migrated.Publisher115ID)
	}
}

func TestQshareForwardRequestRelay(t *testing.T) {
	a := testQshareApp(t)
	createActiveQshareLicense(t, a, "owner@example.com")
	createActiveQshareLicense(t, a, "receiver@example.com")
	createActiveQshareLicense(t, a, "worker@example.com")
	r := testQshareRouter(a)

	publish := qshareRequest(t, r, "/api/qshare/resources/publish", sampleQsharePublishBody("owner@example.com", "qmby-owner", 0, "Interstellar"))
	if publish.Code != http.StatusOK {
		t.Fatalf("publish code = %d, body = %s", publish.Code, publish.Body.String())
	}
	var published struct {
		Resource qshareResourceResponse `json:"resource"`
	}
	if err := json.Unmarshal(publish.Body.Bytes(), &published); err != nil {
		t.Fatalf("decode publish: %v", err)
	}
	fileID := published.Resource.Files[0].ID

	createBody := `{"email":"receiver@example.com","instance_id":"qmby-receiver","beijing_time":"2026-06-09 17:30:00","resource_id":` + strconvUint(published.Resource.ID) + `,"file_ids":[` + strconvUint(fileID) + `],"target_115_id":"123456"}`
	created := qshareRequest(t, r, "/api/qshare/forward/request", createBody)
	if created.Code != http.StatusOK {
		t.Fatalf("forward create code = %d, body = %s", created.Code, created.Body.String())
	}

	ownerPollWithout115ID := qshareRequest(t, r, "/api/qshare/forward/poll", qshareBaseBody("owner@example.com", "qmby-owner", false))
	if ownerPollWithout115ID.Code != http.StatusOK || !strings.Contains(ownerPollWithout115ID.Body.String(), `"requests":[]`) {
		t.Fatalf("poll without publisher 115 ids should be empty: code = %d body = %s", ownerPollWithout115ID.Code, ownerPollWithout115ID.Body.String())
	}

	poll := qshareRequest(t, r, "/api/qshare/forward/poll", qshareForwardPollBody("worker@example.com", "qmby-worker", "4577361"))
	if poll.Code != http.StatusOK || !strings.Contains(poll.Body.String(), `"chat_mid":"mid-1"`) || !strings.Contains(poll.Body.String(), `"publisher_115_id":"4577361"`) {
		t.Fatalf("forward poll failed: code = %d body = %s", poll.Code, poll.Body.String())
	}

	var polled struct {
		Requests []qshareForwardRequestResponse `json:"requests"`
	}
	if err := json.Unmarshal(poll.Body.Bytes(), &polled); err != nil || len(polled.Requests) != 1 {
		t.Fatalf("decode forward poll: err = %v body = %s", err, poll.Body.String())
	}
	completeBody := `{"email":"worker@example.com","instance_id":"qmby-worker","beijing_time":"2026-06-09 17:30:00","request_id":` + strconvUint(polled.Requests[0].ID) + `,"publisher_115_ids":["4577361"]}`
	completed := qshareRequest(t, r, "/api/qshare/forward/complete", completeBody)
	if completed.Code != http.StatusOK {
		t.Fatalf("forward complete failed: code = %d body = %s", completed.Code, completed.Body.String())
	}
	emptyPoll := qshareRequest(t, r, "/api/qshare/forward/poll", qshareForwardPollBody("worker@example.com", "qmby-worker", "4577361"))
	if emptyPoll.Code != http.StatusOK || !strings.Contains(emptyPoll.Body.String(), `"requests":[]`) {
		t.Fatalf("completed request should not be polled again: code = %d body = %s", emptyPoll.Code, emptyPoll.Body.String())
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
	r.POST("/api/qshare/status", a.requireLicenseKey(), a.qshareStatus)
	r.POST("/api/qshare/resources/publish", a.requireLicenseKey(), a.publishQshareResource)
	r.POST("/api/qshare/resources/list", a.requireLicenseKey(), a.listQshareResources)
	r.POST("/api/qshare/resources/detail", a.requireLicenseKey(), a.getQshareResourceDetail)
	r.POST("/api/qshare/resources/delete", a.requireLicenseKey(), a.deleteQshareResource)
	r.POST("/api/qshare/forward/request", a.requireLicenseKey(), a.createQshareForwardRequests)
	r.POST("/api/qshare/forward/poll", a.requireLicenseKey(), a.pollQshareForwardRequests)
	r.POST("/api/qshare/forward/complete", a.requireLicenseKey(), a.completeQshareForwardRequest)
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

func qshareRequest(t *testing.T, r *gin.Engine, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-key")
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	return res
}

func qshareBaseBody(email, instanceID string, publishFolderConfigured bool) string {
	return `{"email":"` + email + `","instance_id":"` + instanceID + `","beijing_time":"2026-06-09 17:30:00","publish_folder_configured":` + strconv.FormatBool(publishFolderConfigured) + `}`
}

func qshareResourceIDBody(email, instanceID string, publishFolderConfigured bool, resourceID uint) string {
	return `{"email":"` + email + `","instance_id":"` + instanceID + `","beijing_time":"2026-06-09 17:30:00","publish_folder_configured":` + strconv.FormatBool(publishFolderConfigured) + `,"resource_id":` + strconvUint(resourceID) + `}`
}

func qshareForwardPollBody(email, instanceID string, publisher115IDs ...string) string {
	ids, _ := json.Marshal(publisher115IDs)
	return `{"email":"` + email + `","instance_id":"` + instanceID + `","beijing_time":"2026-06-09 17:30:00","publish_folder_configured":false,"publisher_115_ids":` + string(ids) + `}`
}

func sampleQsharePublishBody(email, instanceID string, resourceID uint, title string) string {
	return sampleQsharePublishBodyWithSource(email, instanceID, resourceID, title, "115://Movies/Interstellar")
}

func sampleQsharePublishBodyWithSource(email, instanceID string, resourceID uint, title, sourcePath string) string {
	idField := ""
	if resourceID > 0 {
		idField = `"id":` + strconvUint(resourceID) + `,`
	}
	return `{
		"email": "` + email + `",
		"instance_id": "` + instanceID + `",
		"beijing_time": "2026-06-09 17:30:00",
		"publish_folder_configured": true,
		"resource": {
			` + idField + `
			"media_type": "movie",
			"tmdb_id": 157336,
			"title": "` + title + `",
			"year": 2014,
			"poster_url": "https://image.tmdb.org/t/p/w500/poster.jpg",
			"source_path": "` + sourcePath + `",
			"publisher_115_id": "4577361",
			"file_count": 1,
			"total_size": 123456789,
			"files": [{
				"id": "local-file-id",
				"name": "Interstellar.mkv",
				"relative_path": "Interstellar/Interstellar.mkv",
				"size": 123456789,
				"sha1": "0123456789abcdef0123456789abcdef01234567"
				,"chat_mid": "mid-1"
				,"chat_contact_id": "1182480"
			}]
		}
	}`
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

func strconvUint(id uint) string {
	return strconv.FormatUint(uint64(id), 10)
}

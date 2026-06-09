package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	qshareStatusPublished = "published"
	qshareStatusCanceled  = "canceled"
)

type qshareIdentity struct {
	Email      string
	InstanceID string
}

type qshareResourceRequest struct {
	Email      string              `json:"email"`
	InstanceID string              `json:"instance_id"`
	Title      string              `json:"title"`
	MediaType  string              `json:"media_type"`
	TMDBID     string              `json:"tmdb_id"`
	Year       int                 `json:"year"`
	PosterURL  string              `json:"poster_url"`
	Files      []qshareFileRequest `json:"files"`
}

type qshareFileRequest struct {
	Name          string `json:"name"`
	Size          int64  `json:"size"`
	SHA1          string `json:"sha1"`
	RelativePath  string `json:"relative_path"`
	SeasonNumber  *int   `json:"season_number"`
	EpisodeNumber *int   `json:"episode_number"`
}

type qshareResourceSummary struct {
	ID        uint      `json:"id"`
	SourceID  string    `json:"source_id"`
	Title     string    `json:"title"`
	MediaType string    `json:"media_type"`
	TMDBID    string    `json:"tmdb_id"`
	Year      int       `json:"year"`
	PosterURL string    `json:"poster_url"`
	FileCount int       `json:"file_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type qshareResourceDetail struct {
	qshareResourceSummary
	Files []QshareFile `json:"files"`
}

func (a *app) listQshareResources(c *gin.Context) {
	identity, ok := a.qshareIdentityFromQuery(c)
	if !ok {
		return
	}
	if !a.requireQshareMember(c, identity) || !a.requireQshareContributor(c, identity) {
		return
	}

	limit := parsePositiveInt(c.DefaultQuery("limit", "50"), 50, 100)
	offset := parsePositiveInt(c.DefaultQuery("offset", "0"), 0, 1000000)
	tx := a.db.Model(&QshareResource{}).Where("status = ?", qshareStatusPublished)
	if mediaType := strings.TrimSpace(c.Query("media_type")); mediaType != "" {
		tx = tx.Where("media_type = ?", truncate(mediaType, 32))
	}
	if tmdbID := strings.TrimSpace(c.Query("tmdb_id")); tmdbID != "" {
		tx = tx.Where("tmdb_id = ?", truncate(tmdbID, 64))
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var resources []QshareResource
	if err := tx.Order("updated_at DESC, id DESC").Limit(limit).Offset(offset).Find(&resources).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	summaries, err := a.qshareSummaries(resources)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"resources": summaries, "total": total})
}

func (a *app) getQshareResource(c *gin.Context) {
	identity, ok := a.qshareIdentityFromQuery(c)
	if !ok {
		return
	}
	if !a.requireQshareMember(c, identity) || !a.requireQshareContributor(c, identity) {
		return
	}

	resource, ok := a.loadPublishedQshareResource(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{"resource": qshareDetail(resource)})
}

func (a *app) publishQshareResource(c *gin.Context) {
	var req qshareResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "bad request"})
		return
	}
	identity, ok := qshareIdentityFromRequest(c, req)
	if !ok || !a.requireQshareMember(c, identity) {
		return
	}
	resource, files, ok := normalizeQshareResourceRequest(c, req)
	if !ok {
		return
	}
	resource.PublisherEmail = identity.Email
	resource.InstanceID = identity.InstanceID
	resource.SourceID = qshareSourceID(identity.Email, identity.InstanceID)
	resource.Status = qshareStatusPublished

	if err := a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&resource).Error; err != nil {
			return err
		}
		for i := range files {
			files[i].ResourceID = resource.ID
		}
		return tx.Create(&files).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	resource.Files = files
	c.JSON(http.StatusOK, gin.H{"success": true, "resource": qshareDetail(resource)})
}

func (a *app) updateQshareResource(c *gin.Context) {
	var req qshareResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "bad request"})
		return
	}
	identity, ok := qshareIdentityFromRequest(c, req)
	if !ok || !a.requireQshareMember(c, identity) {
		return
	}
	resource, files, ok := normalizeQshareResourceRequest(c, req)
	if !ok {
		return
	}

	id, ok := qshareResourceID(c)
	if !ok {
		return
	}
	if err := a.db.Transaction(func(tx *gorm.DB) error {
		var existing QshareResource
		if err := tx.Where("id = ? AND publisher_email = ? AND instance_id = ? AND status = ?", id, identity.Email, identity.InstanceID, qshareStatusPublished).First(&existing).Error; err != nil {
			return err
		}
		updates := map[string]any{
			"title":      resource.Title,
			"media_type": resource.MediaType,
			"tmdb_id":    resource.TMDBID,
			"year":       resource.Year,
			"poster_url": resource.PosterURL,
			"updated_at": time.Now().In(beijingLocation()),
		}
		if err := tx.Model(&existing).Updates(updates).Error; err != nil {
			return err
		}
		if err := tx.Where("resource_id = ?", existing.ID).Delete(&QshareFile{}).Error; err != nil {
			return err
		}
		for i := range files {
			files[i].ResourceID = existing.ID
		}
		if err := tx.Create(&files).Error; err != nil {
			return err
		}
		resource = existing
		return tx.Preload("Files").First(&resource, existing.ID).Error
	}); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "resource not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "resource": qshareDetail(resource)})
}

func (a *app) cancelQshareResource(c *gin.Context) {
	identity, ok := a.qshareIdentityFromQuery(c)
	if !ok {
		return
	}
	if !a.requireQshareMember(c, identity) {
		return
	}
	id, ok := qshareResourceID(c)
	if !ok {
		return
	}
	result := a.db.Model(&QshareResource{}).
		Where("id = ? AND publisher_email = ? AND instance_id = ? AND status = ?", id, identity.Email, identity.InstanceID, qshareStatusPublished).
		Updates(map[string]any{"status": qshareStatusCanceled, "updated_at": time.Now().In(beijingLocation())})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "resource not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (a *app) qshareIdentityFromQuery(c *gin.Context) (qshareIdentity, bool) {
	identity := qshareIdentity{
		Email:      normalizeEmail(c.Query("email")),
		InstanceID: truncate(strings.TrimSpace(c.Query("instance_id")), 128),
	}
	if !validEmail(identity.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid email"})
		return identity, false
	}
	if identity.InstanceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "instance_id required"})
		return identity, false
	}
	return identity, true
}

func qshareIdentityFromRequest(c *gin.Context, req qshareResourceRequest) (qshareIdentity, bool) {
	identity := qshareIdentity{
		Email:      normalizeEmail(req.Email),
		InstanceID: truncate(strings.TrimSpace(req.InstanceID), 128),
	}
	if !validEmail(identity.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid email"})
		return identity, false
	}
	if identity.InstanceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "instance_id required"})
		return identity, false
	}
	return identity, true
}

func (a *app) requireQshareMember(c *gin.Context, identity qshareIdentity) bool {
	member, err := a.activeQshareMember(identity.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return false
	}
	if !member {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "active license required"})
		return false
	}
	return true
}

func (a *app) activeQshareMember(email string) (bool, error) {
	now := time.Now().In(beijingLocation())
	var code ActivationCode
	if err := a.db.Where("email = ? AND status = ?", email, StatusActive).
		Order("created_at DESC, id DESC").
		First(&code).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	if code.ExpiresAt != nil && !code.ExpiresAt.After(now) {
		return false, a.db.Model(&code).Update("status", StatusExpired).Error
	}
	return true, nil
}

func (a *app) requireQshareContributor(c *gin.Context, identity qshareIdentity) bool {
	var count int64
	err := a.db.Model(&QshareResource{}).
		Joins("JOIN qshare_files ON qshare_files.resource_id = qshare_resources.id").
		Where("qshare_resources.publisher_email = ? AND qshare_resources.instance_id = ? AND qshare_resources.status = ?", identity.Email, identity.InstanceID, qshareStatusPublished).
		Group("qshare_resources.id").
		Count(&count).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return false
	}
	if count == 0 {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "publish one valid resource before browsing qshare"})
		return false
	}
	return true
}

func normalizeQshareResourceRequest(c *gin.Context, req qshareResourceRequest) (QshareResource, []QshareFile, bool) {
	resource := QshareResource{
		Title:     truncate(strings.TrimSpace(req.Title), 256),
		MediaType: truncate(strings.TrimSpace(req.MediaType), 32),
		TMDBID:    truncate(strings.TrimSpace(req.TMDBID), 64),
		Year:      req.Year,
		PosterURL: truncate(strings.TrimSpace(req.PosterURL), 1024),
	}
	if resource.Title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "title required"})
		return resource, nil, false
	}
	if resource.MediaType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "media_type required"})
		return resource, nil, false
	}
	if resource.TMDBID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "tmdb_id required"})
		return resource, nil, false
	}
	if resource.Year <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "year required"})
		return resource, nil, false
	}
	if resource.PosterURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "poster_url required"})
		return resource, nil, false
	}
	files := make([]QshareFile, 0, len(req.Files))
	for _, item := range req.Files {
		file := QshareFile{
			Name:          truncate(strings.TrimSpace(item.Name), 512),
			Size:          item.Size,
			SHA1:          strings.ToLower(truncate(strings.TrimSpace(item.SHA1), 40)),
			RelativePath:  truncate(strings.TrimSpace(item.RelativePath), 1024),
			SeasonNumber:  item.SeasonNumber,
			EpisodeNumber: item.EpisodeNumber,
		}
		if file.Name == "" || file.Size <= 0 || file.SHA1 == "" || file.RelativePath == "" {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid file metadata"})
			return resource, nil, false
		}
		files = append(files, file)
	}
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "files required"})
		return resource, nil, false
	}
	return resource, files, true
}

func (a *app) loadPublishedQshareResource(c *gin.Context) (QshareResource, bool) {
	id, ok := qshareResourceID(c)
	if !ok {
		return QshareResource{}, false
	}
	var resource QshareResource
	if err := a.db.Preload("Files").Where("id = ? AND status = ?", id, qshareStatusPublished).First(&resource).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "resource not found"})
			return resource, false
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return resource, false
	}
	return resource, true
}

func qshareResourceID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid resource id"})
		return 0, false
	}
	return uint(id), true
}

func (a *app) qshareSummaries(resources []QshareResource) ([]qshareResourceSummary, error) {
	summaries := make([]qshareResourceSummary, 0, len(resources))
	if len(resources) == 0 {
		return summaries, nil
	}
	ids := make([]uint, 0, len(resources))
	for _, resource := range resources {
		ids = append(ids, resource.ID)
	}
	var counts []struct {
		ResourceID uint
		Count      int
	}
	if err := a.db.Model(&QshareFile{}).
		Select("resource_id, COUNT(*) AS count").
		Where("resource_id IN ?", ids).
		Group("resource_id").
		Find(&counts).Error; err != nil {
		return nil, err
	}
	countMap := make(map[uint]int, len(counts))
	for _, count := range counts {
		countMap[count.ResourceID] = count.Count
	}
	for _, resource := range resources {
		summary := qshareSummary(resource)
		summary.FileCount = countMap[resource.ID]
		summaries = append(summaries, summary)
	}
	return summaries, nil
}

func qshareDetail(resource QshareResource) qshareResourceDetail {
	return qshareResourceDetail{
		qshareResourceSummary: qshareSummary(resource),
		Files:                 resource.Files,
	}
}

func qshareSummary(resource QshareResource) qshareResourceSummary {
	return qshareResourceSummary{
		ID:        resource.ID,
		SourceID:  resource.SourceID,
		Title:     resource.Title,
		MediaType: resource.MediaType,
		TMDBID:    resource.TMDBID,
		Year:      resource.Year,
		PosterURL: resource.PosterURL,
		FileCount: len(resource.Files),
		CreatedAt: resource.CreatedAt,
		UpdatedAt: resource.UpdatedAt,
	}
}

func qshareSourceID(email, instanceID string) string {
	sum := sha256.Sum256([]byte(normalizeEmail(email) + "|" + strings.TrimSpace(instanceID)))
	return "source_" + hex.EncodeToString(sum[:])[:12]
}

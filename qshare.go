package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	qshareStatusPublished = "published"
	qshareStatusDeleted   = "deleted"
)

type qshareIdentity struct {
	Email      string
	InstanceID string
}

type qshareBaseRequest struct {
	Email                   string `json:"email"`
	InstanceID              string `json:"instance_id"`
	BeijingTime             string `json:"beijing_time"`
	PublishFolderConfigured bool   `json:"publish_folder_configured"`
}

type qsharePublishRequest struct {
	Email                   string                `json:"email"`
	InstanceID              string                `json:"instance_id"`
	BeijingTime             string                `json:"beijing_time"`
	PublishFolderConfigured bool                  `json:"publish_folder_configured"`
	Resource                qshareResourcePayload `json:"resource"`
}

type qshareResourceIDRequest struct {
	Email                   string `json:"email"`
	InstanceID              string `json:"instance_id"`
	BeijingTime             string `json:"beijing_time"`
	PublishFolderConfigured bool   `json:"publish_folder_configured"`
	ResourceID              any    `json:"resource_id"`
}

type qshareResourcePayload struct {
	ID         any                 `json:"id"`
	MediaType  string              `json:"media_type"`
	TMDBID     any                 `json:"tmdb_id"`
	Title      string              `json:"title"`
	Year       int                 `json:"year"`
	PosterURL  string              `json:"poster_url"`
	SourcePath string              `json:"source_path"`
	FileCount  int                 `json:"file_count"`
	TotalSize  int64               `json:"total_size"`
	Files      []qshareFilePayload `json:"files"`
}

type qshareFilePayload struct {
	ID            any    `json:"id"`
	Name          string `json:"name"`
	RelativePath  string `json:"relative_path"`
	Size          int64  `json:"size"`
	SHA1          string `json:"sha1"`
	SeasonNumber  *int   `json:"season_number,omitempty"`
	EpisodeNumber *int   `json:"episode_number,omitempty"`
}

type qshareResourceResponse struct {
	ID         uint                 `json:"id"`
	OwnerLabel string               `json:"owner_label"`
	MediaType  string               `json:"media_type"`
	TMDBID     string               `json:"tmdb_id"`
	Title      string               `json:"title"`
	Year       int                  `json:"year"`
	PosterURL  string               `json:"poster_url"`
	SourcePath string               `json:"source_path"`
	FileCount  int                  `json:"file_count"`
	TotalSize  int64                `json:"total_size"`
	Files      []qshareFileResponse `json:"files,omitempty"`
	CreatedAt  time.Time            `json:"created_at"`
	UpdatedAt  time.Time            `json:"updated_at"`
}

type qshareFileResponse struct {
	ID            uint   `json:"id"`
	Name          string `json:"name"`
	RelativePath  string `json:"relative_path"`
	Size          int64  `json:"size"`
	SHA1          string `json:"sha1"`
	SeasonNumber  *int   `json:"season_number,omitempty"`
	EpisodeNumber *int   `json:"episode_number,omitempty"`
}

func (a *app) qshareStatus(c *gin.Context) {
	var req qshareBaseRequest
	if !qshareBindJSON(c, &req) {
		return
	}
	identity, ok := qshareIdentityFromFields(c, req.Email, req.InstanceID, req.BeijingTime)
	if !ok {
		return
	}
	enabled, err := a.activeQshareMember(identity.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	sharedCount, resourceCount, err := a.qshareCounts(identity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"enabled":        enabled,
		"can_browse":     enabled && req.PublishFolderConfigured,
		"shared_count":   sharedCount,
		"resource_count": resourceCount,
	})
}

func (a *app) publishQshareResource(c *gin.Context) {
	var req qsharePublishRequest
	if !qshareBindJSON(c, &req) {
		return
	}
	identity, ok := qshareIdentityFromFields(c, req.Email, req.InstanceID, req.BeijingTime)
	if !ok || !a.requireQshareMember(c, identity) {
		return
	}
	resource, files, ok := normalizeQshareResourcePayload(c, req.Resource)
	if !ok {
		return
	}
	resource.PublisherEmail = identity.Email
	resource.InstanceID = identity.InstanceID
	resource.Status = qshareStatusPublished

	if err := a.db.Transaction(func(tx *gorm.DB) error {
		existing, found, err := findExistingQshareResource(tx, identity, resource, qsharePayloadUint(req.Resource.ID))
		if err != nil {
			return err
		}
		if found {
			resource.ID = existing.ID
			resource.CreatedAt = existing.CreatedAt
			if err := tx.Model(&existing).Updates(map[string]any{
				"media_type":  resource.MediaType,
				"tmdb_id":     resource.TMDBID,
				"title":       resource.Title,
				"year":        resource.Year,
				"poster_url":  resource.PosterURL,
				"source_path": resource.SourcePath,
				"file_count":  resource.FileCount,
				"total_size":  resource.TotalSize,
				"status":      qshareStatusPublished,
				"updated_at":  time.Now().In(beijingLocation()),
			}).Error; err != nil {
				return err
			}
			if err := tx.Where("resource_id = ?", existing.ID).Delete(&QshareFile{}).Error; err != nil {
				return err
			}
		} else if err := tx.Create(&resource).Error; err != nil {
			return err
		}
		for i := range files {
			files[i].ResourceID = resource.ID
		}
		if err := tx.Create(&files).Error; err != nil {
			return err
		}
		return tx.Preload("Files").First(&resource, resource.ID).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"resource": qshareResourceResponseFromModel(resource, true)})
}

func (a *app) listQshareResources(c *gin.Context) {
	var req qshareBaseRequest
	if !qshareBindJSON(c, &req) {
		return
	}
	identity, ok := qshareIdentityFromFields(c, req.Email, req.InstanceID, req.BeijingTime)
	if !ok {
		return
	}
	enabled, err := a.activeQshareMember(identity.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !enabled || !req.PublishFolderConfigured {
		c.JSON(http.StatusOK, gin.H{"can_browse": false, "resources": []qshareResourceResponse{}})
		return
	}

	var resources []QshareResource
	if err := a.db.Where("status = ?", qshareStatusPublished).Order("updated_at DESC, id DESC").Find(&resources).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response := make([]qshareResourceResponse, 0, len(resources))
	for _, resource := range resources {
		response = append(response, qshareResourceResponseFromModel(resource, false))
	}
	c.JSON(http.StatusOK, gin.H{"can_browse": true, "resources": response})
}

func (a *app) getQshareResourceDetail(c *gin.Context) {
	var req qshareResourceIDRequest
	if !qshareBindJSON(c, &req) {
		return
	}
	identity, ok := qshareIdentityFromFields(c, req.Email, req.InstanceID, req.BeijingTime)
	if !ok || !a.requireQshareMember(c, identity) || !requireQsharePublishFolder(c, req.PublishFolderConfigured) {
		return
	}
	if req.ResourceID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "resource_id required"})
		return
	}
	var resource QshareResource
	resourceID := qsharePayloadUint(req.ResourceID)
	if resourceID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "resource_id required"})
		return
	}
	if err := a.db.Preload("Files").Where("id = ? AND status = ?", resourceID, qshareStatusPublished).First(&resource).Error; err != nil {
		qshareNotFound(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"resource": qshareResourceResponseFromModel(resource, true)})
}

func (a *app) deleteQshareResource(c *gin.Context) {
	var req qshareResourceIDRequest
	if !qshareBindJSON(c, &req) {
		return
	}
	identity, ok := qshareIdentityFromFields(c, req.Email, req.InstanceID, req.BeijingTime)
	if !ok || !a.requireQshareMember(c, identity) {
		return
	}
	if req.ResourceID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "resource_id required"})
		return
	}
	result := a.db.Model(&QshareResource{}).
		Where("id = ? AND publisher_email = ? AND instance_id = ? AND status = ?", qsharePayloadUint(req.ResourceID), identity.Email, identity.InstanceID, qshareStatusPublished).
		Updates(map[string]any{"status": qshareStatusDeleted, "updated_at": time.Now().In(beijingLocation())})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func qshareBindJSON(c *gin.Context, req any) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return false
	}
	return true
}

func qshareIdentityFromFields(c *gin.Context, email, instanceID, beijingTime string) (qshareIdentity, bool) {
	identity := qshareIdentity{
		Email:      normalizeEmail(email),
		InstanceID: truncate(strings.TrimSpace(instanceID), 128),
	}
	if !validEmail(identity.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email"})
		return identity, false
	}
	if identity.InstanceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "instance_id required"})
		return identity, false
	}
	if parseBeijingTime(beijingTime, beijingLocation()).IsZero() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid beijing_time"})
		return identity, false
	}
	return identity, true
}

func (a *app) requireQshareMember(c *gin.Context, identity qshareIdentity) bool {
	member, err := a.activeQshareMember(identity.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return false
	}
	if !member {
		c.JSON(http.StatusForbidden, gin.H{"error": "active license required"})
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

func (a *app) qshareCounts(identity qshareIdentity) (int64, int64, error) {
	var sharedCount int64
	if err := a.db.Model(&QshareResource{}).
		Where("publisher_email = ? AND instance_id = ? AND status = ? AND file_count > 0", identity.Email, identity.InstanceID, qshareStatusPublished).
		Count(&sharedCount).Error; err != nil {
		return 0, 0, err
	}
	var resourceCount int64
	if err := a.db.Model(&QshareResource{}).Where("status = ?", qshareStatusPublished).Count(&resourceCount).Error; err != nil {
		return 0, 0, err
	}
	return sharedCount, resourceCount, nil
}

func requireQsharePublishFolder(c *gin.Context, configured bool) bool {
	if configured {
		return true
	}
	c.JSON(http.StatusForbidden, gin.H{"error": "publish_folder_configured required"})
	return false
}

func normalizeQshareResourcePayload(c *gin.Context, payload qshareResourcePayload) (QshareResource, []QshareFile, bool) {
	resource := QshareResource{
		MediaType:  truncate(strings.TrimSpace(payload.MediaType), 32),
		TMDBID:     truncate(qshareTMDBIDString(payload.TMDBID), 64),
		Title:      truncate(strings.TrimSpace(payload.Title), 256),
		Year:       payload.Year,
		PosterURL:  truncate(strings.TrimSpace(payload.PosterURL), 1024),
		SourcePath: truncate(strings.TrimSpace(payload.SourcePath), 1024),
		FileCount:  payload.FileCount,
		TotalSize:  payload.TotalSize,
	}
	if resource.MediaType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "media_type required"})
		return resource, nil, false
	}
	if resource.Title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title required"})
		return resource, nil, false
	}
	if resource.Year <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "year required"})
		return resource, nil, false
	}
	if resource.PosterURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "poster_url required"})
		return resource, nil, false
	}
	if resource.SourcePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source_path required"})
		return resource, nil, false
	}

	files := make([]QshareFile, 0, len(payload.Files))
	var totalSize int64
	for _, item := range payload.Files {
		file := QshareFile{
			Name:          truncate(strings.TrimSpace(item.Name), 512),
			RelativePath:  truncate(strings.TrimSpace(item.RelativePath), 1024),
			Size:          item.Size,
			SHA1:          strings.ToUpper(truncate(strings.TrimSpace(item.SHA1), 40)),
			SeasonNumber:  item.SeasonNumber,
			EpisodeNumber: item.EpisodeNumber,
		}
		if file.Name == "" || file.Size <= 0 || file.SHA1 == "" || file.RelativePath == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file metadata"})
			return resource, nil, false
		}
		totalSize += file.Size
		files = append(files, file)
	}
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "files required"})
		return resource, nil, false
	}
	if resource.FileCount <= 0 {
		resource.FileCount = len(files)
	}
	if resource.TotalSize <= 0 {
		resource.TotalSize = totalSize
	}
	if resource.FileCount != len(files) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file_count mismatch"})
		return resource, nil, false
	}
	return resource, files, true
}

func findExistingQshareResource(tx *gorm.DB, identity qshareIdentity, resource QshareResource, id uint) (QshareResource, bool, error) {
	var existing QshareResource
	if id > 0 {
		err := tx.Where("id = ? AND publisher_email = ? AND instance_id = ?", id, identity.Email, identity.InstanceID).First(&existing).Error
		if err == nil {
			return existing, true, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return existing, false, err
		}
	}
	tx = tx.Where("publisher_email = ? AND instance_id = ? AND media_type = ?", identity.Email, identity.InstanceID, resource.MediaType)
	if resource.TMDBID != "" {
		tx = tx.Where("tmdb_id = ?", resource.TMDBID)
	} else {
		tx = tx.Where("title = ? AND source_path = ?", resource.Title, resource.SourcePath)
	}
	err := tx.First(&existing).Error
	if err == nil {
		return existing, true, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return existing, false, nil
	}
	return existing, false, err
}

func qshareTMDBIDString(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return strings.TrimSpace(fmt.Sprint(v))
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func qsharePayloadUint(value any) uint {
	switch v := value.(type) {
	case nil:
		return 0
	case uint:
		return v
	case int:
		if v > 0 {
			return uint(v)
		}
	case int64:
		if v > 0 {
			return uint(v)
		}
	case float64:
		if v > 0 && v == float64(uint64(v)) {
			return uint(v)
		}
	case string:
		n, err := strconv.ParseUint(strings.TrimSpace(v), 10, 64)
		if err == nil {
			return uint(n)
		}
	}
	return 0
}

func qshareResourceResponseFromModel(resource QshareResource, includeFiles bool) qshareResourceResponse {
	response := qshareResourceResponse{
		ID:         resource.ID,
		OwnerLabel: qshareOwnerLabel(resource.PublisherEmail, resource.InstanceID),
		MediaType:  resource.MediaType,
		TMDBID:     resource.TMDBID,
		Title:      resource.Title,
		Year:       resource.Year,
		PosterURL:  resource.PosterURL,
		SourcePath: resource.SourcePath,
		FileCount:  resource.FileCount,
		TotalSize:  resource.TotalSize,
		CreatedAt:  resource.CreatedAt,
		UpdatedAt:  resource.UpdatedAt,
	}
	if includeFiles {
		response.Files = make([]qshareFileResponse, 0, len(resource.Files))
		for _, file := range resource.Files {
			response.Files = append(response.Files, qshareFileResponse{
				ID:            file.ID,
				Name:          file.Name,
				RelativePath:  file.RelativePath,
				Size:          file.Size,
				SHA1:          file.SHA1,
				SeasonNumber:  file.SeasonNumber,
				EpisodeNumber: file.EpisodeNumber,
			})
		}
	}
	return response
}

func qshareOwnerLabel(email, instanceID string) string {
	sum := sha256.Sum256([]byte(normalizeEmail(email) + "|" + strings.TrimSpace(instanceID)))
	return "Qshare-" + strings.ToUpper(hex.EncodeToString(sum[:])[:8])
}

func qshareNotFound(c *gin.Context, err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

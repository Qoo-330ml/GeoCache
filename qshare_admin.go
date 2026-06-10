package main

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type adminQshareResource struct {
	QshareResource
	PublisherEmail string `json:"publisher_email"`
	InstanceID     string `json:"instance_id"`
	OwnerLabel     string `json:"owner_label"`
}

type adminQshareUpdateRequest struct {
	MediaType  string `json:"media_type"`
	TMDBID     string `json:"tmdb_id"`
	Title      string `json:"title"`
	Year       int    `json:"year"`
	PosterURL  string `json:"poster_url"`
	SourcePath string `json:"source_path"`
}

func (a *app) listAdminQshareResources(c *gin.Context) {
	limit := parsePositiveInt(c.DefaultQuery("limit", "100"), 100, 200)
	offset := parsePositiveInt(c.DefaultQuery("offset", "0"), 0, 1000000)
	tx := a.db.Model(&QshareResource{})
	if search := strings.TrimSpace(c.Query("search")); search != "" {
		like := "%" + strings.ToLower(search) + "%"
		tx = tx.Where("LOWER(title) LIKE ? OR LOWER(tmdb_id) LIKE ? OR LOWER(publisher_email) LIKE ? OR LOWER(instance_id) LIKE ? OR LOWER(source_path) LIKE ?", like, like, like, like, like)
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		tx = tx.Where("status = ?", status)
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
	response := make([]adminQshareResource, 0, len(resources))
	for _, resource := range resources {
		response = append(response, adminQshareResource{
			QshareResource: resource,
			PublisherEmail: resource.PublisherEmail,
			InstanceID:     resource.InstanceID,
			OwnerLabel:     qshareOwnerLabel(resource.PublisherEmail, resource.InstanceID),
		})
	}
	c.JSON(http.StatusOK, gin.H{"resources": response, "total": total})
}

func (a *app) getAdminQshareResource(c *gin.Context) {
	id, ok := adminQshareResourceID(c)
	if !ok {
		return
	}
	var resource QshareResource
	if err := a.db.Preload("Files").First(&resource, id).Error; err != nil {
		adminQshareNotFound(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"resource": adminQshareResource{
		QshareResource: resource,
		PublisherEmail: resource.PublisherEmail,
		InstanceID:     resource.InstanceID,
		OwnerLabel:     qshareOwnerLabel(resource.PublisherEmail, resource.InstanceID),
	}})
}

func (a *app) updateAdminQshareResource(c *gin.Context) {
	id, ok := adminQshareResourceID(c)
	if !ok {
		return
	}
	var req adminQshareUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}
	req.MediaType = truncate(strings.TrimSpace(req.MediaType), 32)
	req.TMDBID = truncate(strings.TrimSpace(req.TMDBID), 64)
	req.Title = truncate(strings.TrimSpace(req.Title), 256)
	req.PosterURL = truncate(strings.TrimSpace(req.PosterURL), 1024)
	req.SourcePath = truncate(strings.TrimSpace(req.SourcePath), 1024)
	if req.MediaType == "" || req.Title == "" || req.Year <= 0 || req.PosterURL == "" || req.SourcePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resource metadata"})
		return
	}
	result := a.db.Model(&QshareResource{}).Where("id = ?", id).Updates(map[string]any{
		"media_type":  req.MediaType,
		"tmdb_id":     req.TMDBID,
		"title":       req.Title,
		"year":        req.Year,
		"poster_url":  req.PosterURL,
		"source_path": req.SourcePath,
		"updated_at":  time.Now().In(beijingLocation()),
	})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
		return
	}
	a.getAdminQshareResource(c)
}

func (a *app) unpublishAdminQshareResource(c *gin.Context) {
	id, ok := adminQshareResourceID(c)
	if !ok {
		return
	}
	result := a.db.Model(&QshareResource{}).Where("id = ?", id).Updates(map[string]any{
		"status":     qshareStatusDeleted,
		"updated_at": time.Now().In(beijingLocation()),
	})
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

func adminQshareResourceID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resource id"})
		return 0, false
	}
	return uint(id), true
}

func adminQshareNotFound(c *gin.Context, err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

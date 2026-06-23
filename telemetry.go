package main

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const activeClientWindow = 6 * time.Hour

type geoReportPayload struct {
	IP          string   `json:"ip"`
	Location    string   `json:"location"`
	District    string   `json:"district"`
	Street      string   `json:"street"`
	ISP         string   `json:"isp"`
	Latitude    *float64 `json:"latitude"`
	Longitude   *float64 `json:"longitude"`
	Provider    string   `json:"provider"`
	QmbyVersion string   `json:"qmby_version"`
	BeijingTime string   `json:"beijing_time"`
	InstanceID  string   `json:"instance_id"`
	Email       string   `json:"email"`
}

type licenseVerifyRequest struct {
	ActivationCode string `json:"activation_code"`
	BeijingTime    string `json:"beijing_time"`
	InstanceID     string `json:"instance_id"`
	QmbyVersion    string `json:"qmby_version"`
	EmbyServer     string `json:"emby_server"`
}

func (a *app) telemetryStats(c *gin.Context) {
	now := time.Now().In(beijingLocation())
	activeSince := now.Add(-activeClientWindow)

	var installed, active, activeMembers, checks24h, reports int64
	if err := a.db.Model(&ClientInstall{}).Count(&installed).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := a.db.Model(&ClientInstall{}).Where("last_seen_at >= ?", activeSince).Count(&active).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := a.db.Model(&ClientInstall{}).Where("member = ? AND last_seen_at >= ?", true, activeSince).Count(&activeMembers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := a.db.Model(&LicenseCheck{}).Where("created_at >= ?", now.Add(-24*time.Hour)).Count(&checks24h).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := a.db.Model(&IPReport{}).Count(&reports).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"installed_clients":     installed,
		"active_clients":        active,
		"active_members":        activeMembers,
		"license_checks_24h":    checks24h,
		"ip_reports":            reports,
		"active_window_seconds": int(activeClientWindow.Seconds()),
		"server_beijing_time":   now,
	})
}

func (a *app) listClients(c *gin.Context) {
	limit := parsePositiveInt(c.DefaultQuery("limit", "100"), 100, 200)
	offset := parsePositiveInt(c.DefaultQuery("offset", "0"), 0, 1000000)

	tx := a.db.Model(&ClientInstall{})
	if search := strings.TrimSpace(c.Query("search")); search != "" {
		like := "%" + strings.ToLower(search) + "%"
		tx = tx.Where("LOWER(email) LIKE ? OR LOWER(instance_id) LIKE ? OR last_ip LIKE ?", like, like, "%"+search+"%")
	}
	if active := strings.TrimSpace(c.Query("active")); active == "1" || strings.EqualFold(active, "true") {
		tx = tx.Where("last_seen_at >= ?", time.Now().In(beijingLocation()).Add(-activeClientWindow))
	}
	if member := strings.TrimSpace(c.Query("member")); member == "1" || strings.EqualFold(member, "true") {
		tx = tx.Where("member = ?", true)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var clients []ClientInstall
	if err := tx.Order("last_seen_at DESC, id DESC").Limit(limit).Offset(offset).Find(&clients).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"clients":               clients,
		"total":                 total,
		"active_window_seconds": int(activeClientWindow.Seconds()),
	})
}

func (a *app) reportIP(c *gin.Context) {
	var req geoReportPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}
	if req.IP == "" {
		req.IP = c.ClientIP()
	}
	req = normalizeGeoReport(req)
	if net.ParseIP(req.IP) == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ip"})
		return
	}
	if req.QmbyVersion == "" {
		req.QmbyVersion = "Qmby"
	}
	if err := a.db.Transaction(func(tx *gorm.DB) error {
		if err := createIPReport(tx, c, req); err != nil {
			return err
		}
		if err := upsertIPBest(tx, req); err != nil {
			return err
		}
		return updateClientGeo(tx, c, req)
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (a *app) lookupIP(c *gin.Context) {
	ip := strings.TrimSpace(c.Query("ip"))
	if net.ParseIP(ip) == nil {
		c.JSON(http.StatusBadRequest, gin.H{"found": false, "ip": ip, "error": "invalid ip"})
		return
	}
	var best IPBest
	if err := a.db.First(&best, "ip = ?", ip).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusOK, gin.H{"found": false, "ip": ip})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"found": false, "ip": ip, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"found":      true,
		"ip":         best.IP,
		"location":   best.Location,
		"district":   best.District,
		"street":     best.Street,
		"isp":        best.ISP,
		"latitude":   best.Latitude,
		"longitude":  best.Longitude,
		"count":      best.Count,
		"updated_at": best.UpdatedAt,
	})
}

func normalizeGeoReport(p geoReportPayload) geoReportPayload {
	p.IP = strings.TrimSpace(p.IP)
	p.Location = truncate(strings.TrimSpace(p.Location), 256)
	p.District = truncate(strings.TrimSpace(p.District), 128)
	p.Street = truncate(strings.TrimSpace(p.Street), 256)
	p.ISP = truncate(strings.TrimSpace(p.ISP), 128)
	p.Provider = truncate(firstNonEmpty(strings.TrimSpace(p.Provider), "unknown"), 64)
	p.QmbyVersion = truncate(strings.TrimSpace(p.QmbyVersion), 64)
	p.BeijingTime = strings.TrimSpace(p.BeijingTime)
	p.InstanceID = truncate(strings.TrimSpace(p.InstanceID), 128)
	p.Email = normalizeEmail(p.Email)
	return p
}

func clientSeenPayload(req licenseVerifyRequest, c *gin.Context) geoReportPayload {
	return normalizeGeoReport(geoReportPayload{
		IP:          c.ClientIP(),
		InstanceID:  req.InstanceID,
		QmbyVersion: req.QmbyVersion,
		Provider:    "license",
	})
}

func recordClientSeen(tx *gorm.DB, c *gin.Context, req licenseVerifyRequest, code *ActivationCode, member bool, status string, serverNow, clientTime time.Time, seen geoReportPayload) error {
	email := ""
	if code != nil {
		email = normalizeEmail(code.Email)
	}
	instanceID := truncate(strings.TrimSpace(req.InstanceID), 128)
	version := truncate(strings.TrimSpace(req.QmbyVersion), 64)
	embyServer := truncate(strings.TrimSpace(req.EmbyServer), 128)
	ip := firstNonEmpty(seen.IP, c.ClientIP())
	key := clientKey(email, instanceID, ip)
	if key == "" {
		return nil
	}

	client := ClientInstall{
		ClientKey:             key,
		Email:                 email,
		Status:                truncate(status, 32),
		Member:                member,
		InstanceID:            instanceID,
		QmbyVersion:           version,
		EmbyServer:            embyServer,
		FirstSeenAt:           serverNow,
		LastSeenAt:            serverNow,
		LastClientBeijingTime: &clientTime,
		FirstIP:               ip,
		LastIP:                ip,
		ReportCount:           1,
		UserAgent:             truncate(c.Request.UserAgent(), 256),
	}
	if code != nil {
		client.ActivationCodeID = code.ID
		client.Level = code.Level
	}

	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "client_key"}},
		DoUpdates: clause.Assignments(map[string]any{
			"activation_code_id":       client.ActivationCodeID,
			"email":                    client.Email,
			"level":                    client.Level,
			"status":                   client.Status,
			"member":                   client.Member,
			"instance_id":              client.InstanceID,
			"qmby_version":             client.QmbyVersion,
			"emby_server":              client.EmbyServer,
			"last_seen_at":             client.LastSeenAt,
			"last_client_beijing_time": client.LastClientBeijingTime,
			"last_ip":                  client.LastIP,
			"report_count":             gorm.Expr("report_count + 1"),
			"user_agent":               client.UserAgent,
			"updated_at":               serverNow,
		}),
	}).Create(&client).Error
}

func updateClientGeo(tx *gorm.DB, c *gin.Context, geo geoReportPayload) error {
	key := clientKey(geo.Email, geo.InstanceID, geo.IP)
	if key == "" {
		return nil
	}
	updates := map[string]any{
		"last_ip":    firstNonEmpty(geo.IP, c.ClientIP()),
		"location":   geo.Location,
		"district":   geo.District,
		"street":     geo.Street,
		"isp":        geo.ISP,
		"latitude":   geo.Latitude,
		"longitude":  geo.Longitude,
		"provider":   geo.Provider,
		"updated_at": time.Now().In(beijingLocation()),
	}
	if geo.QmbyVersion != "" {
		updates["qmby_version"] = geo.QmbyVersion
	}
	return tx.Model(&ClientInstall{}).Where("client_key = ?", key).Updates(updates).Error
}

func createIPReport(tx *gorm.DB, c *gin.Context, geo geoReportPayload) error {
	report := IPReport{
		IP:            geo.IP,
		Location:      geo.Location,
		District:      geo.District,
		Street:        geo.Street,
		ISP:           geo.ISP,
		Latitude:      geo.Latitude,
		Longitude:     geo.Longitude,
		ClientVersion: geo.QmbyVersion,
		InstanceID:    geo.InstanceID,
		Email:         geo.Email,
		UserAgent:     truncate(c.Request.UserAgent(), 256),
	}
	return tx.Create(&report).Error
}

func upsertIPBest(tx *gorm.DB, geo geoReportPayload) error {
	best := IPBest{
		IP:        geo.IP,
		Location:  geo.Location,
		District:  geo.District,
		Street:    geo.Street,
		ISP:       geo.ISP,
		Latitude:  geo.Latitude,
		Longitude: geo.Longitude,
		Count:     1,
	}
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "ip"}},
		DoUpdates: clause.Assignments(map[string]any{
			"location":   best.Location,
			"district":   best.District,
			"street":     best.Street,
			"isp":        best.ISP,
			"latitude":   best.Latitude,
			"longitude":  best.Longitude,
			"count":      gorm.Expr("count + 1"),
			"updated_at": time.Now().In(beijingLocation()),
		}),
	}).Create(&best).Error
}

func clientKey(email, instanceID, ip string) string {
	if strings.TrimSpace(instanceID) != "" {
		return "instance:" + truncate(strings.TrimSpace(instanceID), 128)
	}
	if strings.TrimSpace(email) != "" && strings.TrimSpace(ip) != "" {
		return "email-ip:" + normalizeEmail(email) + "|" + strings.TrimSpace(ip)
	}
	if strings.TrimSpace(email) != "" {
		return "email:" + normalizeEmail(email)
	}
	if strings.TrimSpace(ip) != "" {
		return "ip:" + strings.TrimSpace(ip)
	}
	return ""
}

func (a *app) listIPBests(c *gin.Context) {
	limit := parsePositiveInt(c.DefaultQuery("limit", "100"), 100, 200)
	offset := parsePositiveInt(c.DefaultQuery("offset", "0"), 0, 1000000)

	tx := a.db.Model(&IPBest{})
	if search := strings.TrimSpace(c.Query("search")); search != "" {
		like := "%" + strings.ToLower(search) + "%"
		tx = tx.Where("LOWER(ip) LIKE ? OR LOWER(location) LIKE ? OR LOWER(district) LIKE ? OR LOWER(street) LIKE ? OR LOWER(isp) LIKE ?", like, like, like, like, like)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var bests []IPBest
	if err := tx.Order("count DESC, updated_at DESC, id DESC").Limit(limit).Offset(offset).Find(&bests).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"bests": bests,
		"total": total,
	})
}

func parsePositiveInt(value string, fallback, max int) int {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || n < 0 {
		return fallback
	}
	if max > 0 && n > max {
		return max
	}
	return n
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

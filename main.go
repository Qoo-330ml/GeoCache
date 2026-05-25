package main

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type app struct {
	db            *gorm.DB
	adminUser     string
	adminPassword string
	licenseAPIKey string
}

func main() {
	gin.SetMode(gin.ReleaseMode)

	db, err := openDatabase(env("DB_PATH", "/data/license.db"))
	if err != nil {
		log.Fatal(err)
	}

	a := &app{
		db:            db,
		adminUser:     env("ADMIN_USER", "admin"),
		adminPassword: env("ADMIN_PASSWORD", "change-me"),
		licenseAPIKey: strings.TrimSpace(os.Getenv("LICENSE_API_KEY")),
	}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.GET("/", func(c *gin.Context) { c.Redirect(http.StatusFound, "/admin") })
	r.GET("/admin", serveAdmin)
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	r.POST("/api/license/verify", a.requireLicenseKey(), a.verifyLicense)
	r.POST("/api/ip/report", a.requireLicenseKey(), a.reportIP)
	r.GET("/api/ip/lookup", a.lookupIP)

	admin := r.Group("/api/admin")
	admin.Use(a.basicAuth())
	admin.GET("/codes", a.listCodes)
	admin.POST("/codes", a.createCode)
	admin.POST("/codes/:id/disable", a.disableCode)
	admin.GET("/stats", a.telemetryStats)
	admin.GET("/clients", a.listClients)
	admin.GET("/ip-bests", a.listIPBests)

	addr := env("SERVER_ADDR", ":2090")
	log.Printf("qmby-license-server listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}

func (a *app) basicAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, password, ok := c.Request.BasicAuth()
		if !ok ||
			subtle.ConstantTimeCompare([]byte(user), []byte(a.adminUser)) != 1 ||
			subtle.ConstantTimeCompare([]byte(password), []byte(a.adminPassword)) != 1 {
			c.Header("WWW-Authenticate", `Basic realm="Qmby License Admin"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}

func (a *app) requireLicenseKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.licenseAPIKey == "" {
			c.Next()
			return
		}
		token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		if token == "" {
			token = c.GetHeader("X-License-Key")
		}
		if token == "" {
			token = c.GetHeader("X-API-Key")
		}
		if subtle.ConstantTimeCompare([]byte(token), []byte(a.licenseAPIKey)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"member": false, "error": "invalid license api key"})
			return
		}
		c.Next()
	}
}

func (a *app) listCodes(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit <= 0 || limit > 200 {
		limit = 100
	}

	tx := a.db.Model(&ActivationCode{})
	if search := strings.TrimSpace(c.Query("search")); search != "" {
		like := "%" + strings.ToLower(search) + "%"
		tx = tx.Where("LOWER(email) LIKE ? OR code_prefix LIKE ? OR note LIKE ?", like, like, like)
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		tx = tx.Where("status = ?", status)
	}
	if level := strings.TrimSpace(c.Query("level")); level != "" {
		tx = tx.Where("level = ?", level)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var codes []ActivationCode
	if err := tx.Order("created_at DESC, id DESC").Limit(limit).Offset(offset).Find(&codes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var clientCounts []struct {
		ActivationCodeID uint
		ClientCount      int64
	}
	a.db.Model(&ClientInstall{}).
		Select("activation_code_id, COUNT(DISTINCT client_key) AS client_count").
		Where("activation_code_id > 0").
		Group("activation_code_id").
		Find(&clientCounts)

	countMap := make(map[uint]int64)
	for _, cc := range clientCounts {
		countMap[cc.ActivationCodeID] = cc.ClientCount
	}

	c.JSON(http.StatusOK, gin.H{"codes": codes, "total": total, "client_counts": countMap})
}

func (a *app) createCode(c *gin.Context) {
	var req struct {
		Email string `json:"email"`
		Level string `json:"level"`
		Note  string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}
	email := normalizeEmail(req.Email)
	if !validEmail(email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email"})
		return
	}
	level := strings.ToLower(strings.TrimSpace(req.Level))
	durationDays, ok := levelDurationDays[level]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid level"})
		return
	}

	var existing int64
	if err := a.db.Model(&ActivationCode{}).
		Where("email = ? AND status IN ?", email, []string{StatusIssued, StatusActive}).
		Count(&existing).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if existing > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "email already has an available license"})
		return
	}

	plainCode, err := generateCode()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "generate code failed"})
		return
	}
	code := &ActivationCode{
		CodeHash:     hashCode(plainCode),
		CodePrefix:   codePrefix(plainCode),
		Email:        email,
		Level:        level,
		DurationDays: durationDays,
		Status:       StatusIssued,
		Note:         strings.TrimSpace(req.Note),
	}
	if err := a.db.Create(code).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": code, "plain_code": plainCode})
}

func (a *app) disableCode(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := a.db.Model(&ActivationCode{}).Where("id = ?", id).Update("status", StatusDisabled).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "disabled"})
}

func (a *app) verifyLicense(c *gin.Context) {
	var req licenseVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"member": false, "error": "bad request"})
		return
	}
	email := normalizeEmail(req.Email)
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"member": false, "error": "email required"})
		return
	}

	loc := beijingLocation()
	serverNow := time.Now().In(loc)
	clientTime := parseBeijingTime(req.BeijingTime, loc)
	if clientTime.IsZero() {
		clientTime = serverNow
	}
	seen := clientSeenPayload(req, c)

	var code ActivationCode
	member := false
	err := a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("email = ? AND status IN ?", email, []string{StatusIssued, StatusActive}).
			Order("CASE WHEN status = 'active' THEN 0 ELSE 1 END, created_at DESC, id DESC").
			First(&code).Error; err != nil {
			return err
		}

		if code.ExpiresAt != nil && !code.ExpiresAt.After(serverNow) {
			code.Status = StatusExpired
			if err := tx.Save(&code).Error; err != nil {
				return err
			}
		}

		if code.Status == StatusIssued {
			code.Status = StatusActive
			code.StartsAt = &serverNow
			code.FirstSeenAt = &serverNow
			code.FirstIP = c.ClientIP()
			code.ExpiresAt = expiryFor(code.DurationDays, serverNow)
		}

		code.LastSeenAt = &serverNow
		code.LastClientBeijingTime = &clientTime
		code.LastInstanceID = strings.TrimSpace(req.InstanceID)
		code.LastQmbyVersion = strings.TrimSpace(req.QmbyVersion)
		code.LastIP = c.ClientIP()
		code.VerifyCount++
		member = code.Status == StatusActive && (code.ExpiresAt == nil || code.ExpiresAt.After(serverNow))
		if err := tx.Save(&code).Error; err != nil {
			return err
		}

		check := LicenseCheck{
			ActivationCodeID:  code.ID,
			Email:             email,
			Level:             code.Level,
			Status:            code.Status,
			Member:            member,
			InstanceID:        strings.TrimSpace(req.InstanceID),
			QmbyVersion:       strings.TrimSpace(req.QmbyVersion),
			ClientBeijingTime: clientTime,
			ServerBeijingTime: serverNow,
			IP:                c.ClientIP(),
			UserAgent:         truncate(c.Request.UserAgent(), 256),
		}
		if err := tx.Create(&check).Error; err != nil {
			return err
		}
		return recordClientSeen(tx, c, req, &code, member, code.Status, serverNow, clientTime, seen)
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			_ = a.db.Transaction(func(tx *gorm.DB) error {
				check := LicenseCheck{
					Email:             email,
					Status:            "none",
					Member:            false,
					InstanceID:        strings.TrimSpace(req.InstanceID),
					QmbyVersion:       strings.TrimSpace(req.QmbyVersion),
					ClientBeijingTime: clientTime,
					ServerBeijingTime: serverNow,
					IP:                c.ClientIP(),
					UserAgent:         truncate(c.Request.UserAgent(), 256),
				}
				if err := tx.Create(&check).Error; err != nil {
					return err
				}
				return recordClientSeen(tx, c, req, nil, false, "none", serverNow, clientTime, seen)
			})
			c.JSON(http.StatusOK, gin.H{"member": false, "status": "none", "server_beijing_time": serverNow})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"member": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"member":              member,
		"level":               code.Level,
		"level_label":         levelDisplayNames[code.Level],
		"status":              code.Status,
		"starts_at":           code.StartsAt,
		"expires_at":          code.ExpiresAt,
		"server_beijing_time": serverNow,
	})
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validEmail(email string) bool {
	return strings.Contains(email, "@") && strings.Contains(email, ".")
}

func generateCode() (string, error) {
	buf := make([]byte, 15)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	raw := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf)
	raw = strings.ToUpper(raw[:20])
	return "QMBY-" + raw[:4] + "-" + raw[4:8] + "-" + raw[8:12] + "-" + raw[12:16] + "-" + raw[16:20], nil
}

func hashCode(code string) string {
	normalized := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(code), " ", ""))
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

func codePrefix(code string) string {
	parts := strings.Split(code, "-")
	if len(parts) >= 3 {
		return parts[0] + "-" + parts[1] + "-" + parts[2]
	}
	if len(code) <= 14 {
		return code
	}
	return code[:14]
}

func expiryFor(days int, startsAt time.Time) *time.Time {
	if days <= 0 {
		return nil
	}
	expiresAt := startsAt.AddDate(0, 0, days)
	return &expiresAt
}

func parseBeijingTime(value string, loc *time.Location) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05"} {
		var (
			t   time.Time
			err error
		)
		if layout == time.RFC3339 {
			t, err = time.Parse(layout, value)
		} else {
			t, err = time.ParseInLocation(layout, value, loc)
		}
		if err == nil {
			return t.In(loc)
		}
	}
	return time.Time{}
}

func beijingLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("CST", 8*60*60)
	}
	return loc
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

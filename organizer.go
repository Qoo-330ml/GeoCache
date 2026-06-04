package main

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	organizerKindFailed           = "failed"
	organizerKindManualFailed     = "manual_failed"
	organizerKindManualReorganize = "manual_reorganize"
)

type organizerFailedRecordsRequest struct {
	Email       string `json:"email"`
	BeijingTime string `json:"beijing_time"`
	InstanceID  string `json:"instance_id"`
	QmbyVersion string `json:"qmby_version"`
	Records     string `json:"records"`
}

func (a *app) submitOrganizerFailedRecords(c *gin.Context) {
	var req organizerFailedRecordsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "bad request"})
		return
	}

	loc := beijingLocation()
	email := normalizeEmail(req.Email)
	clientTime := parseBeijingTime(req.BeijingTime, loc)
	instanceID := truncate(strings.TrimSpace(req.InstanceID), 128)
	version := truncate(strings.TrimSpace(req.QmbyVersion), 64)
	recordsText := strings.TrimSpace(req.Records)
	if !validEmail(email) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid email"})
		return
	}
	if clientTime.IsZero() {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid beijing_time"})
		return
	}
	if instanceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "instance_id required"})
		return
	}
	if recordsText == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "records required"})
		return
	}

	lines := nonEmptyLines(req.Records)
	submission := OrganizerFailedRecordSubmission{
		Email:             email,
		ClientBeijingTime: clientTime,
		InstanceID:        instanceID,
		QmbyVersion:       version,
		RawRecords:        req.Records,
		RecordCount:       len(lines),
		IP:                c.ClientIP(),
		UserAgent:         truncate(c.Request.UserAgent(), 256),
	}

	if err := a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&submission).Error; err != nil {
			return err
		}
		rows := parseOrganizerFailedRecordRows(lines, submission.ID, email, instanceID, version)
		if len(rows) == 0 {
			return nil
		}
		return tx.Create(&rows).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "整理失败记录已提交", "count": len(lines)})
}

func (a *app) exportOrganizerFailedRecords(c *gin.Context) {
	var records []OrganizerFailedRecord
	if err := a.db.Order("created_at DESC, submission_id DESC, line_number ASC").Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	filename := "organizer_failed_records_" + time.Now().In(beijingLocation()).Format("20060102_150405") + ".csv"
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Writer.WriteHeader(http.StatusOK)
	_, _ = c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})

	w := csv.NewWriter(c.Writer)
	_ = w.Write([]string{
		"received_at",
		"submission_id",
		"line_number",
		"email",
		"instance_id",
		"qmby_version",
		"event_time",
		"kind",
		"original_name",
		"original_path",
		"message",
		"parse_error",
		"raw_line",
	})
	for _, record := range records {
		eventTime := ""
		if record.EventTime != nil {
			eventTime = record.EventTime.Format(time.RFC3339)
		}
		_ = w.Write([]string{
			record.CreatedAt.In(beijingLocation()).Format("2006-01-02 15:04:05"),
			strconv.FormatUint(uint64(record.SubmissionID), 10),
			strconv.Itoa(record.LineNumber),
			record.Email,
			record.InstanceID,
			record.QmbyVersion,
			eventTime,
			record.Kind,
			record.OriginalName,
			record.OriginalPath,
			record.Message,
			record.ParseError,
			record.RawLine,
		})
	}
	w.Flush()
}

func nonEmptyLines(records string) []string {
	var lines []string
	for _, line := range strings.Split(strings.ReplaceAll(records, "\r\n", "\n"), "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

func parseOrganizerFailedRecordRows(lines []string, submissionID uint, email, instanceID, version string) []OrganizerFailedRecord {
	rows := make([]OrganizerFailedRecord, 0, len(lines))
	for i, line := range lines {
		row := OrganizerFailedRecord{
			SubmissionID: submissionID,
			LineNumber:   i + 1,
			Email:        email,
			InstanceID:   instanceID,
			QmbyVersion:  version,
			RawLine:      line,
		}
		parts := strings.SplitN(line, "\t", 5)
		if len(parts) != 5 {
			row.ParseError = "expected 5 tab-separated fields"
			rows = append(rows, row)
			continue
		}
		eventTime, err := time.Parse(time.RFC3339, strings.TrimSpace(parts[0]))
		if err != nil {
			row.ParseError = "invalid event time"
		} else {
			t := eventTime.In(beijingLocation())
			row.EventTime = &t
		}
		row.Kind = truncate(strings.TrimSpace(parts[1]), 32)
		if !validOrganizerFailedRecordKind(row.Kind) {
			row.ParseError = firstNonEmpty(row.ParseError, "invalid kind")
		}
		row.OriginalName = truncate(strings.TrimSpace(parts[2]), 512)
		row.OriginalPath = truncate(strings.TrimSpace(parts[3]), 1024)
		row.Message = strings.TrimSpace(parts[4])
		rows = append(rows, row)
	}
	return rows
}

func validOrganizerFailedRecordKind(kind string) bool {
	switch kind {
	case organizerKindFailed, organizerKindManualFailed, organizerKindManualReorganize:
		return true
	default:
		return false
	}
}

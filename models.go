package main

import "time"

const (
	LevelTrial     = "trial"
	LevelYearly    = "yearly"
	LevelPermanent = "permanent"
	LevelBeta      = "beta"

	StatusIssued   = "issued"
	StatusActive   = "active"
	StatusDisabled = "disabled"
	StatusExpired  = "expired"
)

var levelDurationDays = map[string]int{
	LevelTrial:     7,
	LevelYearly:    365,
	LevelPermanent: 0,
	LevelBeta:      0,
}

var levelDisplayNames = map[string]string{
	LevelTrial:     "试用会员",
	LevelYearly:    "年费会员",
	LevelPermanent: "永久会员",
	LevelBeta:      "内测会员",
}

type ActivationCode struct {
	ID                    uint       `gorm:"primaryKey" json:"id"`
	CodeHash              string     `gorm:"uniqueIndex;size:64" json:"-"`
	CodePrefix            string     `gorm:"size:20" json:"code_prefix"`
	Email                 string     `gorm:"index;size:320;not null" json:"email"`
	Level                 string     `gorm:"index;size:32;not null" json:"level"`
	DurationDays          int        `gorm:"not null;default:0" json:"duration_days"`
	Status                string     `gorm:"index;size:32;not null;default:issued" json:"status"`
	StartsAt              *time.Time `json:"starts_at"`
	ExpiresAt             *time.Time `gorm:"index" json:"expires_at"`
	FirstSeenAt           *time.Time `json:"first_seen_at"`
	LastSeenAt            *time.Time `gorm:"index" json:"last_seen_at"`
	LastClientBeijingTime *time.Time `json:"last_client_beijing_time"`
	LastInstanceID        string     `gorm:"size:128" json:"last_instance_id"`
	LastQmbyVersion       string     `gorm:"size:64" json:"last_qmby_version"`
	LastEmbyServer        string     `gorm:"size:128" json:"last_emby_server"`
	FirstIP               string     `gorm:"size:64" json:"first_ip"`
	LastIP                string     `gorm:"size:64" json:"last_ip"`
	VerifyCount           int        `gorm:"not null;default:0" json:"verify_count"`
	Note                  string     `gorm:"size:512" json:"note"`
	CreatedAt             time.Time  `gorm:"autoCreateTime;index" json:"created_at"`
	UpdatedAt             time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

type LicenseCheck struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	ActivationCodeID  uint      `gorm:"index" json:"activation_code_id"`
	Email             string    `gorm:"index;size:320" json:"email"`
	Level             string    `gorm:"size:32" json:"level"`
	Status            string    `gorm:"size:32" json:"status"`
	Member            bool      `gorm:"index" json:"member"`
	InstanceID        string    `gorm:"size:128" json:"instance_id"`
	QmbyVersion       string    `gorm:"size:64" json:"qmby_version"`
	EmbyServer        string    `gorm:"size:128" json:"emby_server"`
	ClientBeijingTime time.Time `json:"client_beijing_time"`
	ServerBeijingTime time.Time `gorm:"index" json:"server_beijing_time"`
	IP                string    `gorm:"size:64" json:"ip"`
	Location          string    `gorm:"size:256" json:"location"`
	District          string    `gorm:"size:128" json:"district"`
	Street            string    `gorm:"size:256" json:"street"`
	ISP               string    `gorm:"size:128" json:"isp"`
	UserAgent         string    `gorm:"size:256" json:"user_agent"`
	CreatedAt         time.Time `gorm:"autoCreateTime;index" json:"created_at"`
}

type ClientInstall struct {
	ID                    uint       `gorm:"primaryKey" json:"id"`
	ClientKey             string     `gorm:"uniqueIndex;size:160;not null" json:"client_key"`
	ActivationCodeID      uint       `gorm:"index" json:"activation_code_id"`
	Email                 string     `gorm:"index;size:320" json:"email"`
	Level                 string     `gorm:"size:32" json:"level"`
	Status                string     `gorm:"size:32" json:"status"`
	Member                bool       `gorm:"index" json:"member"`
	InstanceID            string     `gorm:"index;size:128" json:"instance_id"`
	QmbyVersion           string     `gorm:"size:64" json:"qmby_version"`
	EmbyServer            string     `gorm:"size:128" json:"emby_server"`
	FirstSeenAt           time.Time  `gorm:"autoCreateTime;index" json:"first_seen_at"`
	LastSeenAt            time.Time  `gorm:"index" json:"last_seen_at"`
	LastClientBeijingTime *time.Time `json:"last_client_beijing_time"`
	FirstIP               string     `gorm:"size:64" json:"first_ip"`
	LastIP                string     `gorm:"index;size:64" json:"last_ip"`
	Location              string     `gorm:"size:256" json:"location"`
	District              string     `gorm:"size:128" json:"district"`
	Street                string     `gorm:"size:256" json:"street"`
	ISP                   string     `gorm:"size:128" json:"isp"`
	Latitude              *float64   `json:"latitude"`
	Longitude             *float64   `json:"longitude"`
	Provider              string     `gorm:"size:64" json:"provider"`
	ReportCount           int        `gorm:"not null;default:0" json:"report_count"`
	UserAgent             string     `gorm:"size:256" json:"user_agent"`
	CreatedAt             time.Time  `gorm:"autoCreateTime;index" json:"created_at"`
	UpdatedAt             time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

type IPReport struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	IP            string    `gorm:"index:idx_ip_reports_ip_created_at;size:64;not null" json:"ip"`
	Location      string    `gorm:"size:256" json:"location"`
	District      string    `gorm:"size:128" json:"district"`
	Street        string    `gorm:"size:256" json:"street"`
	ISP           string    `gorm:"size:128" json:"isp"`
	Latitude      *float64  `json:"latitude"`
	Longitude     *float64  `json:"longitude"`
	Provider      string    `gorm:"size:64" json:"provider"`
	ClientVersion string    `gorm:"size:64" json:"client_version"`
	InstanceID    string    `gorm:"index;size:128" json:"instance_id"`
	Email         string    `gorm:"index;size:320" json:"email"`
	UserAgent     string    `gorm:"size:256" json:"user_agent"`
	CreatedAt     time.Time `gorm:"autoCreateTime;index:idx_ip_reports_ip_created_at" json:"created_at"`
}

type IPBest struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	IP        string    `gorm:"uniqueIndex;size:64;not null" json:"ip"`
	Location  string    `gorm:"size:256" json:"location"`
	District  string    `gorm:"size:128" json:"district"`
	Street    string    `gorm:"size:256" json:"street"`
	ISP       string    `gorm:"size:128" json:"isp"`
	Latitude  *float64  `json:"latitude"`
	Longitude *float64  `json:"longitude"`
	Provider  string    `gorm:"size:64" json:"provider"`
	Count     int       `gorm:"not null;default:0" json:"count"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

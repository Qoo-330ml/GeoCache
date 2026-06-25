package main

import "time"

const (
	LevelTrial     = "trial"
	LevelYearly    = "yearly"
	LevelPermanent = "permanent"
	LevelPlus      = "plus"
	LevelPro       = "pro"
	LevelBeta      = "beta"

	StatusIssued   = "issued"
	StatusActive   = "active"
	StatusDisabled = "disabled"
	StatusExpired  = "expired"

	OrderStatusPending   = "pending"
	OrderStatusPaid      = "paid"
	OrderStatusFulfilled = "fulfilled"
	OrderStatusActivated = "activated"
	OrderStatusExpired   = "expired"
	OrderStatusFailed    = "failed"

	FeatureAccessFree     = "free"
	FeatureAccessMember   = "member"
	FeatureAccessPlus     = "plus"
	FeatureAccessPro      = "pro"
	FeatureAccessDisabled = "disabled"
)

var levelDurationDays = map[string]int{
	LevelTrial:     7,
	LevelYearly:    365,
	LevelPermanent: 0,
	LevelPlus:      365,
	LevelPro:       365,
	LevelBeta:      0,
}

var levelDisplayNames = map[string]string{
	LevelTrial:     "试用会员",
	LevelYearly:    "年费会员",
	LevelPermanent: "永久 Pro 会员",
	LevelPlus:      "Plus 会员",
	LevelPro:       "Pro 会员",
	LevelBeta:      "内测会员",
}

var allowedActivationDurations = map[int]bool{
	7:   true,
	30:  true,
	365: true,
	0:   true,
}

type FeaturePolicy struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Key       string    `gorm:"uniqueIndex;size:64;not null" json:"key"`
	Label     string    `gorm:"size:64;not null" json:"label"`
	Access    string    `gorm:"size:16;not null;default:member" json:"access"`
	Enabled   bool      `gorm:"not null;default:true" json:"enabled"`
	SortOrder int       `gorm:"not null;default:0" json:"sort_order"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type MailSetting struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Host      string    `gorm:"size:256" json:"host"`
	Port      string    `gorm:"size:16" json:"port"`
	Username  string    `gorm:"size:256" json:"username"`
	Password  string    `gorm:"size:512" json:"-"`
	From      string    `gorm:"size:320" json:"from"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type PaymentSetting struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	PublicBaseURL      string    `gorm:"size:512" json:"public_base_url"`
	XorPayAID          string    `gorm:"size:128" json:"xorpay_aid"`
	XorPaySecret       string    `gorm:"size:512" json:"-"`
	EnabledPayTypes    string    `gorm:"size:64" json:"enabled_pay_types"`
	OrderExpireSeconds int       `gorm:"not null;default:7200" json:"order_expire_seconds"`
	CreatedAt          time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type FeaturePolicyPayload struct {
	Label   string `json:"label"`
	Access  string `json:"access"`
	Enabled bool   `json:"enabled"`
}

var defaultFeaturePolicies = []FeaturePolicy{
	{Key: "account", Label: "号池管理", Access: FeatureAccessFree, Enabled: true, SortOrder: 10},
	{Key: "account_guest", Label: "访客账号", Access: FeatureAccessPro, Enabled: true, SortOrder: 20},
	{Key: "coplay_copy", Label: "同播复制", Access: FeatureAccessPlus, Enabled: true, SortOrder: 30},
	{Key: "strm_task", Label: "Strm 任务", Access: FeatureAccessFree, Enabled: true, SortOrder: 40},
	{Key: "resource_subscription", Label: "资源订阅", Access: FeatureAccessPlus, Enabled: true, SortOrder: 50},
	{Key: "qshare", Label: "Qshare", Access: FeatureAccessPlus, Enabled: true, SortOrder: 60},
	{Key: "upload_monitor", Label: "文件监控", Access: FeatureAccessFree, Enabled: true, SortOrder: 70},
	{Key: "organizer", Label: "识别整理", Access: FeatureAccessFree, Enabled: true, SortOrder: 80},
	{Key: "cloud_organizer", Label: "云端整理", Access: FeatureAccessPro, Enabled: true, SortOrder: 90},
	{Key: "playback_monitor", Label: "播放监控", Access: FeatureAccessPro, Enabled: true, SortOrder: 100},
}

type ActivationCode struct {
	ID                    uint       `gorm:"primaryKey" json:"id"`
	CodeHash              string     `gorm:"uniqueIndex;size:64" json:"-"`
	PlainCode             string     `gorm:"size:64" json:"plain_code,omitempty"`
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

type PaymentOrder struct {
	ID                  uint       `gorm:"primaryKey" json:"id"`
	OrderID             string     `gorm:"uniqueIndex;size:64;not null" json:"order_id"`
	Provider            string     `gorm:"index;size:32;not null" json:"provider"`
	PayType             string     `gorm:"index;size:32;not null" json:"pay_type"`
	PlanKey             string     `gorm:"index;size:64;not null" json:"plan_key"`
	PlanLevel           string     `gorm:"index;size:32;not null" json:"plan_level"`
	DurationDays        int        `gorm:"not null;default:0" json:"duration_days"`
	AmountCents         int        `gorm:"not null" json:"amount_cents"`
	Status              string     `gorm:"index;size:32;not null;default:pending" json:"status"`
	Email               string     `gorm:"index;size:320;not null" json:"email"`
	InstanceID          string     `gorm:"index;size:128;not null" json:"instance_id"`
	ActivationCodeID    *uint      `gorm:"index" json:"activation_code_id"`
	ActivationCodePlain string     `gorm:"size:64" json:"-"`
	ProviderOrderID     string     `gorm:"index;size:128" json:"provider_order_id"`
	ProviderPayload     string     `gorm:"type:text" json:"provider_payload"`
	PayQRCode           string     `gorm:"type:text" json:"pay_qr"`
	PaidAt              *time.Time `gorm:"index" json:"paid_at"`
	FulfilledAt         *time.Time `json:"fulfilled_at"`
	EmailSentAt         *time.Time `json:"email_sent_at"`
	ExpiresAt           time.Time  `gorm:"index" json:"expires_at"`
	LastError           string     `gorm:"size:512" json:"last_error"`
	CreatedAt           time.Time  `gorm:"autoCreateTime;index" json:"created_at"`
	UpdatedAt           time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
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

type OrganizerFailedRecordSubmission struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	Email             string    `gorm:"index;size:320;not null" json:"email"`
	ClientBeijingTime time.Time `gorm:"index" json:"client_beijing_time"`
	InstanceID        string    `gorm:"index;size:128;not null" json:"instance_id"`
	QmbyVersion       string    `gorm:"size:64" json:"qmby_version"`
	RawRecords        string    `gorm:"type:text;not null" json:"raw_records"`
	RecordCount       int       `gorm:"not null;default:0" json:"record_count"`
	IP                string    `gorm:"size:64" json:"ip"`
	UserAgent         string    `gorm:"size:256" json:"user_agent"`
	CreatedAt         time.Time `gorm:"autoCreateTime;index" json:"created_at"`
}

type OrganizerFailedRecord struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	SubmissionID uint       `gorm:"index;not null" json:"submission_id"`
	LineNumber   int        `gorm:"not null" json:"line_number"`
	Email        string     `gorm:"index;size:320;not null" json:"email"`
	InstanceID   string     `gorm:"index;size:128;not null" json:"instance_id"`
	QmbyVersion  string     `gorm:"size:64" json:"qmby_version"`
	EventTime    *time.Time `gorm:"index" json:"event_time"`
	Kind         string     `gorm:"index;size:32" json:"kind"`
	OriginalName string     `gorm:"size:512" json:"original_name"`
	OriginalPath string     `gorm:"size:1024" json:"original_path"`
	Message      string     `gorm:"type:text" json:"message"`
	RawLine      string     `gorm:"type:text;not null" json:"raw_line"`
	ParseError   string     `gorm:"size:256" json:"parse_error"`
	CreatedAt    time.Time  `gorm:"autoCreateTime;index" json:"created_at"`
}

type QshareResource struct {
	ID             uint         `gorm:"primaryKey" json:"id"`
	PublisherEmail string       `gorm:"index;size:320;not null" json:"-"`
	InstanceID     string       `gorm:"index;size:128;not null" json:"-"`
	Publisher115ID string       `gorm:"index;size:32;not null;default:''" json:"publisher_115_id"`
	SourceID       string       `gorm:"index;size:32;not null" json:"source_id"`
	Title          string       `gorm:"size:256;not null" json:"title"`
	MediaType      string       `gorm:"index;size:32;not null" json:"media_type"`
	TMDBID         string       `gorm:"index;size:64;not null" json:"tmdb_id"`
	Year           int          `gorm:"index;not null" json:"year"`
	PosterURL      string       `gorm:"size:1024;not null" json:"poster_url"`
	SourcePath     string       `gorm:"size:1024;not null" json:"source_path"`
	FileCount      int          `gorm:"not null;default:0" json:"file_count"`
	TotalSize      int64        `gorm:"not null;default:0" json:"total_size"`
	Status         string       `gorm:"index;size:32;not null;default:published" json:"status"`
	Files          []QshareFile `gorm:"foreignKey:ResourceID;constraint:OnDelete:CASCADE" json:"files,omitempty"`
	CreatedAt      time.Time    `gorm:"autoCreateTime;index" json:"created_at"`
	UpdatedAt      time.Time    `gorm:"autoUpdateTime;index" json:"updated_at"`
}

type QshareFile struct {
	ID                    uint      `gorm:"primaryKey" json:"id"`
	ResourceID            uint      `gorm:"index;not null" json:"-"`
	Name                  string    `gorm:"size:512;not null" json:"name"`
	Size                  int64     `gorm:"not null" json:"size"`
	SHA1                  string    `gorm:"index;size:40;not null" json:"sha1"`
	RelativePath          string    `gorm:"size:1024;not null" json:"relative_path"`
	Quality               string    `gorm:"size:512;not null;default:''" json:"quality"`
	ChildrenJSON          string    `gorm:"type:text;not null;default:''" json:"-"`
	IsDir                 bool      `gorm:"index;not null;default:false" json:"is_dir"`
	Publisher115ID        string    `gorm:"index;size:32;not null;default:''" json:"publisher_115_id"`
	ChatMID               string    `gorm:"index;size:32;not null;default:''" json:"-"`
	ChatContactID         string    `gorm:"size:32;not null;default:''" json:"-"`
	ChatPartFileIDsJSON   string    `gorm:"type:text;not null;default:''" json:"-"`
	ChatPartFolderIDsJSON string    `gorm:"type:text;not null;default:''" json:"-"`
	SeasonNumber          *int      `json:"season_number,omitempty"`
	EpisodeNumber         *int      `json:"episode_number,omitempty"`
	CreatedAt             time.Time `gorm:"autoCreateTime" json:"created_at"`
}

type QshareForwardRequest struct {
	ID                uint       `gorm:"primaryKey" json:"id"`
	ResourceID        uint       `gorm:"index;not null" json:"resource_id"`
	FileID            uint       `gorm:"index;not null" json:"file_id"`
	PublisherEmail    string     `gorm:"index;size:320;not null" json:"-"`
	PublisherInstance string     `gorm:"index;size:128;not null" json:"-"`
	RequesterEmail    string     `gorm:"index;size:320;not null" json:"-"`
	RequesterInstance string     `gorm:"index;size:128;not null" json:"-"`
	Publisher115ID    string     `gorm:"index;size:32;not null" json:"publisher_115_id"`
	Target115ID       string     `gorm:"size:32;not null" json:"target_115_id"`
	ChatMID           string     `gorm:"size:32;not null" json:"chat_mid"`
	ChatContactID     string     `gorm:"size:32;not null" json:"chat_contact_id"`
	Status            string     `gorm:"index;size:32;not null;default:pending" json:"status"`
	Error             string     `gorm:"size:1024" json:"error,omitempty"`
	ExpiresAt         time.Time  `gorm:"index;not null" json:"expires_at"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
	CreatedAt         time.Time  `gorm:"autoCreateTime;index" json:"created_at"`
	UpdatedAt         time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

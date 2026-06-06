package model

import "time"

// DeviceFingerprint stores browser-side device signals for abuse detection.
type DeviceFingerprint struct {
	BaseModel
	UserID              uint      `gorm:"not null;uniqueIndex:idx_device_fingerprints_user_fingerprint;index:idx_device_fingerprints_user;comment:用户ID"`
	FingerprintID       string    `gorm:"size:64;not null;uniqueIndex:idx_device_fingerprints_user_fingerprint;index:idx_device_fingerprints_fingerprint;comment:稳定指纹ID"`
	ScreenResolution    string    `gorm:"size:50;not null;default:'';comment:屏幕分辨率"`
	ColorDepth          int       `gorm:"not null;default:0;comment:颜色深度"`
	PixelRatio          float64   `gorm:"type:decimal(6,2);not null;default:0;comment:像素倍率"`
	HardwareConcurrency int       `gorm:"not null;default:0;comment:CPU线程数"`
	DeviceMemory        int       `gorm:"not null;default:0;comment:设备内存GB"`
	MaxTouchPoints      int       `gorm:"not null;default:0;comment:最大触控点数"`
	UserAgent           string    `gorm:"type:text;not null;default:'';comment:用户代理"`
	Language            string    `gorm:"size:32;not null;default:'';comment:浏览器语言"`
	Timezone            string    `gorm:"size:80;not null;default:'';comment:时区"`
	Platform            string    `gorm:"size:80;not null;default:'';comment:平台"`
	CanvasHash          string    `gorm:"size:64;not null;default:'';comment:Canvas哈希"`
	WebGLVendor         string    `gorm:"size:255;not null;default:'';comment:WebGL厂商"`
	WebGLRenderer       string    `gorm:"size:255;not null;default:'';comment:WebGL渲染器"`
	FontsHash           string    `gorm:"size:64;not null;default:'';comment:字体哈希"`
	AudioHash           string    `gorm:"size:64;not null;default:'';comment:音频哈希"`
	IPAddress           string    `gorm:"size:64;not null;default:'';index:idx_device_fingerprints_ip;comment:请求IP"`
	TLSFingerprint      string    `gorm:"size:128;not null;default:'';comment:TLS指纹"`
	FirstSeenAt         time.Time `gorm:"not null;comment:首次出现时间"`
	LastSeenAt          time.Time `gorm:"not null;index:idx_device_fingerprints_last_seen;comment:最近出现时间"`
	SeenCount           int       `gorm:"not null;default:1;comment:出现次数"`
}

// TableName 指定表名。
func (DeviceFingerprint) TableName() string {
	return "device_fingerprints"
}

// FingerprintAssociation stores multi-account detections.
type FingerprintAssociation struct {
	BaseModel
	FingerprintID   string     `gorm:"size:64;not null;uniqueIndex:idx_fingerprint_associations_fingerprint;comment:稳定指纹ID"`
	UserIDsJSON     string     `gorm:"type:text;not null;default:'[]';comment:关联用户ID列表JSON"`
	ConfidenceScore float64    `gorm:"type:decimal(4,3);not null;default:0;comment:置信度"`
	RiskLevel       string     `gorm:"size:32;not null;default:'low';index:idx_fingerprint_associations_risk;comment:风险等级"`
	DetectedAt      time.Time  `gorm:"not null;index:idx_fingerprint_associations_detected;comment:检测时间"`
	IgnoredAt       *time.Time `gorm:"comment:忽略时间"`
	Reason          string     `gorm:"type:text;not null;default:'';comment:处理原因"`
}

// TableName 指定表名。
func (FingerprintAssociation) TableName() string {
	return "fingerprint_associations"
}

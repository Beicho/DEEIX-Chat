package model

import "time"

// CheckInRecord records a user's daily check-in reward.
type CheckInRecord struct {
	BaseModel
	UserID               uint      `gorm:"not null;uniqueIndex:idx_billing_checkins_user_date,priority:1;index:idx_billing_checkins_user_id;comment:用户ID"`
	CheckInDate          time.Time `gorm:"type:date;not null;uniqueIndex:idx_billing_checkins_user_date,priority:2;index:idx_billing_checkins_date;comment:签到日期"`
	RewardNanousd        int64     `gorm:"not null;default:0;comment:奖励金额(纳美元)"`
	ConsecutiveDays      int       `gorm:"not null;default:1;comment:连续签到天数"`
	BalanceTransactionID uint      `gorm:"not null;default:0;index:idx_billing_checkins_balance_tx_id;comment:余额流水ID"`
	RefNo                string    `gorm:"size:128;not null;default:'';index:idx_billing_checkins_ref_no;comment:签到流水号"`
}

func (CheckInRecord) TableName() string { return "billing_checkin_records" }

// ExternalAccountLink stores a binding to an external platform account.
type ExternalAccountLink struct {
	BaseModel
	UserID              uint       `gorm:"not null;uniqueIndex:idx_external_links_user_platform,priority:1;index:idx_external_links_user_id;comment:用户ID"`
	Platform            string     `gorm:"size:32;not null;uniqueIndex:idx_external_links_user_platform,priority:2;uniqueIndex:idx_external_links_platform_external_user,priority:1;comment:外部平台"`
	ExternalUserID      string     `gorm:"size:128;not null;uniqueIndex:idx_external_links_platform_external_user,priority:2;comment:外部用户ID"`
	ExternalDisplayName string     `gorm:"size:128;not null;default:'';comment:外部显示名"`
	LinuxDOSub          string     `gorm:"size:255;not null;default:'';index:idx_external_links_linuxdo_sub;comment:LinuxDo subject"`
	Status              string     `gorm:"size:32;not null;default:'active';index:idx_external_links_status;comment:绑定状态"`
	LinkedAt            time.Time  `gorm:"not null;comment:绑定时间"`
	LastSyncedAt        *time.Time `gorm:"comment:最近同步时间"`
}

func (ExternalAccountLink) TableName() string { return "external_account_links" }

// ExternalTransfer records a one-way external balance transfer.
type ExternalTransfer struct {
	BaseModel
	UserID                uint       `gorm:"not null;index:idx_external_transfers_user_id;comment:用户ID"`
	LinkID                uint       `gorm:"not null;default:0;index:idx_external_transfers_link_id;comment:外部账号绑定ID"`
	Platform              string     `gorm:"size:32;not null;index:idx_external_transfers_platform;comment:外部平台"`
	Direction             string     `gorm:"size:16;not null;default:'in';index:idx_external_transfers_direction;comment:划转方向"`
	ExternalTransferID    string     `gorm:"size:128;not null;default:'';index:idx_external_transfers_external_id;comment:外部划转ID"`
	IdempotencyKey        string     `gorm:"size:128;not null;uniqueIndex:idx_external_transfers_idempotency;comment:幂等键"`
	ExternalAmountUSD     float64    `gorm:"not null;default:0;comment:外部美元面值"`
	CreditedAmountNanousd int64      `gorm:"not null;default:0;comment:本站入账纳美元"`
	BalanceTransactionID  uint       `gorm:"not null;default:0;index:idx_external_transfers_balance_tx_id;comment:余额流水ID"`
	Status                string     `gorm:"size:32;not null;default:'pending';index:idx_external_transfers_status;comment:状态"`
	FailureReason         string     `gorm:"type:text;not null;default:'';comment:失败原因"`
	RequestedAt           time.Time  `gorm:"not null;index:idx_external_transfers_requested_at;comment:请求时间"`
	CompletedAt           *time.Time `gorm:"comment:完成时间"`
}

func (ExternalTransfer) TableName() string { return "external_transfers" }

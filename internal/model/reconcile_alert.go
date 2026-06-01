package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// ReconcileAlert 对账不一致告警记录
type ReconcileAlert struct {
	ID        uint `gorm:"primaryKey"`
	AccountID uint `gorm:"not null;index"`
	UserID    uint `gorm:"not null;index"`

	// 余额与流水累加的差异
	BalanceInDB  decimal.Decimal `gorm:"type:decimal(18,4);not null"` // 数据库记录的可用余额
	BalanceByTxn decimal.Decimal `gorm:"type:decimal(18,4);not null"` // 流水累加得到的余额
	Difference   decimal.Decimal `gorm:"type:decimal(18,4);not null"` // 差额 = InDB - ByTxn

	Status    int8   `gorm:"type:tinyint;not null;default:1;index"` // 1=未处理 2=已处理(假阳/已修复)
	Remark    string `gorm:"type:varchar(255)"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

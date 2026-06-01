package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type WithdrawOrder struct {
	ID        uint   `gorm:"primaryKey"`
	OrderNo   string `gorm:"type:varchar(32);not null;uniqueIndex:uk_order_no"`
	UserID    uint   `gorm:"not null;index"`
	AccountID uint   `gorm:"not null;index"`
	Currency  string `gorm:"type:varchar(4);not null;default:'USD'"`

	Amount decimal.Decimal `gorm:"type:decimal(18,4);not null"`
	Fee    decimal.Decimal `gorm:"type:decimal(18,4);not null;default:0"`

	// 收款信息(学习版明文,生产必须加密)
	PayeeAccount string `gorm:"type:varchar(64);not null"`
	PayeeName    string `gorm:"type:varchar(64);not null"`
	PayeeBank    string `gorm:"type:varchar(64)"` // 可选

	// 状态:1=处理中(已冻结) 2=成功 3=失败已退回
	Status int8 `gorm:"type:tinyint;not null;default:1;index"`

	// 创建时尚未发生的时间和原因,可空
	PaidAt     *time.Time
	FinishedAt *time.Time
	FailReason string `gorm:"type:varchar(255)"` // 仅失败时有值

	Version   uint `gorm:"not null;default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

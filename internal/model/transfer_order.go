package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type TransferOrder struct {
	ID         uint   `gorm:"primaryKey"`
	TransferNo string `gorm:"type:varchar(32);not null;uniqueIndex:uk_transfer_no"` // 幂等核心

	FromUserID    uint            `gorm:"not null;index"`
	FromAccountID uint            `gorm:"not null;index"`
	ToUserID      uint            `gorm:"not null;index"`
	ToAccountID   uint            `gorm:"not null;index"`
	Currency      string          `gorm:"type:varchar(4);not null"`
	Amount        decimal.Decimal `gorm:"type:decimal(18,4);not null"`

	// 转账是同步完成的:开事务 → 扣加 → 提交,要么成功要么失败,没有"处理中"
	// 所以状态机比充值提现简单很多
	Status     int8   `gorm:"type:tinyint;not null;default:2;index"` // 2=成功 3=失败
	FailReason string `gorm:"type:varchar(255)"`
	Remark     string `gorm:"type:varchar(255)"`

	Version   uint `gorm:"not null;default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

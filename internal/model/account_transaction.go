package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type AccountTransaction struct {
	ID            uint            `gorm:"primaryKey"`
	TxnNo         string          `gorm:"type:varchar(32);not null;uniqueIndex:uk_txn_no"`
	AccountID     uint            `gorm:"not null;index"`                                         //哪个账户
	UserID        uint            `gorm:"not null;index"`                                         //哪个用户
	BizType       int8            `gorm:"type:tinyint;not null;uniqueIndex:uk_biz_direction"`     //业务类型 1充值 2提现 3转账
	BizOrderNo    string          `gorm:"type:varchar(32);not null;uniqueIndex:uk_biz_direction"` //关联的业务订单号
	Direction     int8            `gorm:"type:tinyint;not null;uniqueIndex:uk_biz_direction"`     //交易方向 1入账 2出账
	Amount        decimal.Decimal `gorm:"type:decimal(18,4);not null"`
	BalanceBefore decimal.Decimal `gorm:"type:decimal(18,4);not null"`
	BalanceAfter  decimal.Decimal `gorm:"type:decimal(18,4);not null"`
	Currency      string          `gorm:"type:varchar(4);not null"`
	Remark        string          `gorm:"type:varchar(255)"`
	CreatedAt     time.Time
}

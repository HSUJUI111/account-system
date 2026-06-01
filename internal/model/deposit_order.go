package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type DepositOrder struct {
	ID         uint            `gorm:"primaryKey"`
	OrderNo    string          `gorm:"type:varchar(32);not null;uniqueIndex:uk_order_no"` //订单编号
	UserID     uint            `gorm:"not null;index"`                                    //充值用户ID
	AccountID  uint            `gorm:"not null;index"`                                    //入账账户ID                                   //充值用户ID
	Currency   string          `gorm:"type:varchar(4);not null;default:'USD'"`            //币种
	Amount     decimal.Decimal `gorm:"type:decimal(18,4);not null"`                       //充值金额
	Fee        decimal.Decimal `gorm:"type:decimal(18,4);not null;default:0"`             //手续费
	Status     int8            `gorm:"type:tinyint;not null;default:0;index"`             //0待支付 1处理中 2成功 3失败 4已过期
	PaidAt     *time.Time      //付款时间
	FinishedAt *time.Time      //入账时间
	Remark     string          `gorm:"type:varchar(255)"`  //备注
	Version    uint            `gorm:"not null;default:0"` //乐观锁版本号
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

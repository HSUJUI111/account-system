package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type Account struct {
	ID               uint            `gorm:"primaryKey"`
	AccountNo        string          `gorm:"type:varchar(32);not null;uniqueIndex:uk_account_no"`                 //账户编号
	UserID           uint            `gorm:"uniqueIndex:uk_user_currency;not null"`                               //账户所属用户ID
	Currency         string          `gorm:"type:varchar(4);uniqueIndex:uk_user_currency;not null;default:'USD'"` //币种
	AvailableBalance decimal.Decimal `gorm:"type:decimal(18,4);not null;default:0.0000"`                          //可用余额
	FrozenBalance    decimal.Decimal `gorm:"type:decimal(18,4);not null;default:0.0000"`                          //冻结余额
	Status           int8            `gorm:"type:tinyint;default:1"`                                              //账户状态 1正常 2冻结 3销户
	Version          uint            `gorm:"not null;default:0"`                                                  //乐观锁版本号
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (a *Account) TotalBalance() decimal.Decimal {
	return a.AvailableBalance.Add(a.FrozenBalance)
}

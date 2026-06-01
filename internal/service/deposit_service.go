package service

import (
	"account-system/internal/model"
	"account-system/internal/repository"
	"account-system/pkg/idgen"
	"errors"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

var ErrInvalidAmount = errors.New("充值金额必须大于0")

func CreateDepositOrder(userID uint, currency string, amount decimal.Decimal) (*model.DepositOrder, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, ErrInvalidAmount
	}
	var account model.Account

	err := repository.DB.Where("user_id = ? AND currency = ?", userID, currency).First(&account).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrAccountNotFound
	}
	if err != nil {
		return nil, err
	}

	order := &model.DepositOrder{
		OrderNo:   idgen.GenerateNo("DEP_"),
		UserID:    userID,
		AccountID: account.ID, // 关联到刚查到的账户
		Currency:  currency,
		Amount:    amount,
		// Fee 不传,数据库 default 0
		Status: 1, // 1=处理中(创建即进入待回调入账状态)
		// PaidAt / FinishedAt 不赋值,保持 nil(还没付款/入账)
	}

	if err := repository.DB.Create(order).Error; err != nil {
		return nil, err
	}
	return order, nil
}

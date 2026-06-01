package service

import (
	"account-system/internal/model"
	"account-system/internal/repository"
	"account-system/pkg"
	"errors"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

var (
	ErrAccountExists       = errors.New("该用户币种账户已存在")
	ErrAccountNotFound     = errors.New("账户不存在")
	ErrOrderNotFound       = errors.New("订单不存在")
	ErrInvalidPayee        = errors.New("收款信息不完整")
	ErrInsufficientBalance = errors.New("可用余额不足")
)

// 创建账户
func CreateAccount(userID uint, currency string) (*model.Account, error) {
	var exist model.Account
	err := repository.DB.Where("user_id = ? AND currency = ?", userID, currency).First(&exist).Error
	if err == nil {
		return nil, ErrAccountExists
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	account := &model.Account{
		UserID:           userID,
		Currency:         currency,
		AccountNo:        pkg.GenerateNo("ACC_"),
		AvailableBalance: decimal.Zero,
		FrozenBalance:    decimal.Zero,
		Status:           1,
	}

	err = repository.DB.Create(account).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, ErrAccountExists
		}
		return nil, err
	}
	return account, nil

}

func GetAccount(userID uint, currency string) (*model.Account, error) {
	var account model.Account
	err := repository.DB.
		Where("user_id = ? AND currency = ?", userID, currency).
		First(&account).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAccountNotFound
		}
		return nil, err
	}

	return &account, nil
}

// 充值
func ConfirmDeposit(orderNo string) error {
	return repository.DB.Transaction(func(tx *gorm.DB) error {
		var order model.DepositOrder
		if err := tx.Where("order_no = ?", orderNo).First(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrOrderNotFound
			}
			return err
		}
		result := tx.Model(&model.DepositOrder{}).
			Where("order_no = ? AND status = ?", orderNo, 1).
			Updates(map[string]interface{}{
				"status":      2,
				"finished_at": gorm.Expr("NOW()"),
				"version":     gorm.Expr("version + 1"),
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		var account model.Account
		if err := tx.Where("id = ?", order.AccountID).First(&account).Error; err != nil {
			return err
		}
		balanceBefore := account.AvailableBalance
		balanceAfter := balanceBefore.Add(order.Amount)
		if err := tx.Model(&model.Account{}).
			Where("id = ?", order.AccountID).
			Updates(map[string]interface{}{
				"available_balance": balanceAfter,
				"version":           gorm.Expr("version +1"),
			}).Error; err != nil {
			return err
		}
		txn := &model.AccountTransaction{
			TxnNo:         pkg.GenerateNo("TXN_"),
			AccountID:     order.AccountID,
			UserID:        order.UserID,
			BizType:       1,
			BizOrderNo:    order.OrderNo,
			Direction:     1,
			Amount:        order.Amount,
			BalanceBefore: balanceBefore,
			BalanceAfter:  balanceAfter,
			Currency:      order.Currency,
		}
		if err := tx.Create(txn).Error; err != nil {
			return err
		}
		return nil
	})
}

// 提现
func CreateWithdrawOrder(userID uint, currency string, amount decimal.Decimal, payeeAccount string, payeeName string, payeeBank string) (*model.WithdrawOrder, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, ErrInvalidAmount
	}
	if payeeAccount == "" || payeeName == "" {
		return nil, ErrInvalidPayee
	}

	var account model.Account
	err := repository.DB.Where("user_id = ? AND currency = ?", userID, currency).First(&account).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrAccountNotFound
	}
	if err != nil {
		return nil, err
	}
	balanceBefore := account.AvailableBalance
	balanceAfter := balanceBefore.Sub(amount)
	withdraworder := &model.WithdrawOrder{
		OrderNo:      pkg.GenerateNo("WD_"),
		UserID:       userID,
		AccountID:    account.ID,
		Currency:     currency,
		Amount:       amount,
		PayeeAccount: payeeAccount, // ← 之前漏了
		PayeeName:    payeeName,    // ← 之前漏了
		PayeeBank:    payeeBank,
		Status:       1,
	}

	err = repository.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", withdraworder.AccountID).First(&account).Error; err != nil {
			return err
		}
		result := tx.Model(&model.Account{}).
			Where("id = ? AND available_balance >= ? ",
				withdraworder.AccountID, withdraworder.Amount).
			Updates(map[string]interface{}{
				"available_balance": gorm.Expr("available_balance - ?", withdraworder.Amount),
				"frozen_balance":    gorm.Expr("frozen_balance + ?", withdraworder.Amount),
				"version":           gorm.Expr("version + 1"),
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrInsufficientBalance
		}
		if err := tx.Create(withdraworder).Error; err != nil {
			return err
		}
		txn := &model.AccountTransaction{
			TxnNo:         pkg.GenerateNo("TXN_"),
			AccountID:     account.ID,
			UserID:        account.UserID,
			BizType:       2, // 2 = 提现冻结
			BizOrderNo:    withdraworder.OrderNo,
			Direction:     2, // 2 = 支出
			Amount:        amount,
			BalanceBefore: balanceBefore,
			BalanceAfter:  balanceAfter,
			Currency:      currency,
		}
		if err := tx.Create(txn).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return withdraworder, nil
}

// ConfirmWithdraw 模拟代付渠道回调:告知某笔提现订单成功或失败
// success=true  → 代付成功,真正扣减冻结余额
// success=false → 代付失败,把冻结余额解冻退回可用余额
func ConfirmWithdraw(orderNo string, success bool, failReason string) error {
	return repository.DB.Transaction(func(tx *gorm.DB) error {
		// ---- 1. 查订单 ----
		var order model.WithdrawOrder
		if err := tx.Where("order_no = ?", orderNo).First(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrOrderNotFound
			}
			return err
		}

		// 确定目标状态:成功=2,失败已退回=3
		newStatus := int8(2)
		if !success {
			newStatus = 3
		}

		// ---- 2. CAS 改订单状态(状态机幂等)----
		// 只有当前是"1处理中(已冻结)"才能改成成功或失败
		updates := map[string]interface{}{
			"status":      newStatus,
			"finished_at": gorm.Expr("NOW()"),
			"version":     gorm.Expr("version + 1"),
		}
		if success {
			updates["paid_at"] = gorm.Expr("NOW()")
		} else {
			updates["fail_reason"] = failReason
		}

		result := tx.Model(&model.WithdrawOrder{}).
			Where("order_no = ? AND status = ?", orderNo, 1).
			Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			// 状态已经不是"处理中"了,说明此订单已被处理过(重复回调)
			// 幂等:直接返回成功,不重复扣款或退回
			return nil
		}

		// ---- 3. 查账户拿到当前余额,为流水准备 ----
		var account model.Account
		if err := tx.Where("id = ?", order.AccountID).First(&account).Error; err != nil {
			return err
		}

		if success {
			// ===== 代付成功:扣冻结余额(钱真正出账) =====
			// 注意:available_balance 不动!之前冻结时已经从 available 转到 frozen 了
			// 这里只需要从 frozen 扣掉,代表钱真的从总余额里出去了
			if err := tx.Model(&model.Account{}).
				Where("id = ?", order.AccountID).
				Updates(map[string]interface{}{
					"frozen_balance": gorm.Expr("frozen_balance - ?", order.Amount),
					"version":        gorm.Expr("version + 1"),
				}).Error; err != nil {
				return err
			}

			// 记一条"出账流水":BizType=4 提现出账,Direction=2 支出
			// 注意:Before/After 记的是 available 余额(不变),因为可用余额没变
			// 真正变的是 frozen,但我们的流水设计追踪 available。
			// 这是学习版的简化——生产环境会同时追踪 available 和 frozen 的变动。
			txn := &model.AccountTransaction{
				TxnNo:         pkg.GenerateNo("TXN_"),
				AccountID:     account.ID,
				UserID:        account.UserID,
				BizType:       4, // 4 = 提现出账
				BizOrderNo:    order.OrderNo,
				Direction:     2, // 2 = 支出
				Amount:        order.Amount,
				BalanceBefore: account.AvailableBalance,
				BalanceAfter:  account.AvailableBalance, // available 不变
				Currency:      order.Currency,
				Remark:        "提现出账",
			}
			if err := tx.Create(txn).Error; err != nil {
				return err
			}
		} else {
			// ===== 代付失败:解冻退回(钱还给用户) =====
			// frozen 减,available 加,等于把之前冻结时做的操作反向走一遍
			balanceBefore := account.AvailableBalance
			balanceAfter := balanceBefore.Add(order.Amount)

			if err := tx.Model(&model.Account{}).
				Where("id = ?", order.AccountID).
				Updates(map[string]interface{}{
					"available_balance": gorm.Expr("available_balance + ?", order.Amount),
					"frozen_balance":    gorm.Expr("frozen_balance - ?", order.Amount),
					"version":           gorm.Expr("version + 1"),
				}).Error; err != nil {
				return err
			}

			// 记一条"退回流水":BizType=3 提现退回,Direction=1 收入(可用余额增加)
			txn := &model.AccountTransaction{
				TxnNo:         pkg.GenerateNo("TXN_"),
				AccountID:     account.ID,
				UserID:        account.UserID,
				BizType:       3, // 3 = 提现退回
				BizOrderNo:    order.OrderNo,
				Direction:     1, // 1 = 收入
				Amount:        order.Amount,
				BalanceBefore: balanceBefore,
				BalanceAfter:  balanceAfter,
				Currency:      order.Currency,
				Remark:        "提现失败退回",
			}
			if err := tx.Create(txn).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

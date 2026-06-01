package service

import (
	"account-system/internal/model"
	"account-system/internal/repository"

	"github.com/shopspring/decimal"
)

// ReconcileResult 对账结果(单个账户)
type ReconcileResult struct {
	AccountID    uint            `json:"account_id"`
	UserID       uint            `json:"user_id"`
	BalanceInDB  decimal.Decimal `json:"balance_in_db"`  // DB 记录的余额
	BalanceByTxn decimal.Decimal `json:"balance_by_txn"` // 流水累加得到的余额
	Difference   decimal.Decimal `json:"difference"`     // 差额
	IsConsistent bool            `json:"is_consistent"`  // 是否一致
}

// ReconcileAccount 对单个账户进行对账
func ReconcileAccount(accountID uint) (*ReconcileResult, error) {
	// 1. 查账户当前余额
	var account model.Account
	if err := repository.DB.Where("id = ?", accountID).First(&account).Error; err != nil {
		return nil, err
	}

	// 2. 用 SQL 直接求和该账户的所有流水:收入 - 支出
	// 生产做法:加 WHERE created_at <= ? 限定截止时间,避免活数据
	var result struct {
		Income  decimal.Decimal
		Expense decimal.Decimal
	}
	err := repository.DB.Model(&model.AccountTransaction{}).
		Select(`
			COALESCE(SUM(CASE WHEN direction = 1 AND biz_type != 4 THEN amount ELSE 0 END), 0) AS income,
        COALESCE(SUM(CASE WHEN direction = 2 AND biz_type != 4 THEN amount ELSE 0 END), 0) AS expense
		`).
		Where("account_id = ?", accountID).
		Scan(&result).Error
	if err != nil {
		return nil, err
	}

	balanceByTxn := result.Income.Sub(result.Expense)
	balanceInDB := account.AvailableBalance
	diff := balanceInDB.Sub(balanceByTxn)

	res := &ReconcileResult{
		AccountID:    accountID,
		UserID:       account.UserID,
		BalanceInDB:  balanceInDB,
		BalanceByTxn: balanceByTxn,
		Difference:   diff,
		IsConsistent: diff.IsZero(),
	}

	// 3. 不一致 → 写告警表
	if !res.IsConsistent {
		alert := &model.ReconcileAlert{
			AccountID:    accountID,
			UserID:       account.UserID,
			BalanceInDB:  balanceInDB,
			BalanceByTxn: balanceByTxn,
			Difference:   diff,
			Status:       1,
		}
		// 这里故意忽略写告警的错误,因为对账本身的结果更重要;
		// 生产环境通常用日志 + 监控(Prometheus 计数器)双管齐下
		repository.DB.Create(alert)
	}

	return res, nil
}

// ReconcileAll 对所有账户做对账,返回不一致项
func ReconcileAll() ([]*ReconcileResult, error) {
	var accounts []model.Account
	if err := repository.DB.Find(&accounts).Error; err != nil {
		return nil, err
	}

	inconsistent := make([]*ReconcileResult, 0)
	for _, acc := range accounts {
		res, err := ReconcileAccount(acc.ID)
		if err != nil {
			// 单个账户出错不影响其他账户,继续
			continue
		}
		if !res.IsConsistent {
			inconsistent = append(inconsistent, res)
		}
	}

	return inconsistent, nil
}

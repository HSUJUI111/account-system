package service

import (
	"account-system/internal/model"
	"account-system/internal/repository"
	"account-system/pkg"
	"errors"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrSameAccount      = errors.New("不能转账给自己")
	ErrTransferConflict = errors.New("转账请求重复")
)

func clauseForUpdate() clause.Expression {
	return clause.Locking{Strength: "UPDATE"}
}

// Transfer 同步转账:从 fromUser 的 currency 账户,转 amount 到 toUser 的 currency 账户
// transferNo 是客户端生成的幂等键
func Transfer(
	transferNo string,
	fromUserID uint,
	toUserID uint,
	currency string,
	amount decimal.Decimal,
	remark string,
) (*model.TransferOrder, error) {
	// 1. 入参校验
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, ErrInvalidAmount
	}
	if fromUserID == toUserID {
		return nil, ErrSameAccount
	}
	if transferNo == "" {
		transferNo = pkg.GenerateNo("TR_")
	}

	// 2. 幂等检查(事务外,快速失败)
	var existing model.TransferOrder
	err := repository.DB.Where("transfer_no = ?", transferNo).First(&existing).Error
	if err == nil {
		// 已存在,直接返回这条记录(幂等)
		return &existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// 3. 查两个账户(还没加锁,只是先拿到 ID 决定加锁顺序)
	var fromAccount, toAccount model.Account
	if err := repository.DB.Where("user_id = ? AND currency = ?", fromUserID, currency).
		First(&fromAccount).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAccountNotFound
		}
		return nil, err
	}
	if err := repository.DB.Where("user_id = ? AND currency = ?", toUserID, currency).
		First(&toAccount).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAccountNotFound
		}
		return nil, err
	}

	// 4. 决定加锁顺序:小 id 先锁。这是防死锁的核心!
	firstID, secondID := fromAccount.ID, toAccount.ID
	if firstID > secondID {
		firstID, secondID = secondID, firstID
	}

	var order *model.TransferOrder

	err = repository.DB.Transaction(func(tx *gorm.DB) error {
		// 5. 按 id 升序加悲观锁(SELECT FOR UPDATE),锁住两行
		var locked1, locked2 model.Account
		if err := tx.Clauses(clauseForUpdate()).
			Where("id = ?", firstID).First(&locked1).Error; err != nil {
			return err
		}
		if err := tx.Clauses(clauseForUpdate()).
			Where("id = ?", secondID).First(&locked2).Error; err != nil {
			return err
		}

		// 6. 区分哪个是 from、哪个是 to(因为我们按 id 排序锁的,顺序可能跟传参反了)
		var from, to *model.Account
		if locked1.ID == fromAccount.ID {
			from, to = &locked1, &locked2
		} else {
			from, to = &locked2, &locked1
		}

		// 7. 校验余额(已经加锁,可以放心检查)
		if from.AvailableBalance.LessThan(amount) {
			return ErrInsufficientBalance
		}

		// 8. 扣 from
		fromBefore := from.AvailableBalance
		fromAfter := fromBefore.Sub(amount)
		if err := tx.Model(&model.Account{}).
			Where("id = ?", from.ID).
			Updates(map[string]interface{}{
				"available_balance": fromAfter,
				"version":           gorm.Expr("version + 1"),
			}).Error; err != nil {
			return err
		}

		// 9. 加 to
		toBefore := to.AvailableBalance
		toAfter := toBefore.Add(amount)
		if err := tx.Model(&model.Account{}).
			Where("id = ?", to.ID).
			Updates(map[string]interface{}{
				"available_balance": toAfter,
				"version":           gorm.Expr("version + 1"),
			}).Error; err != nil {
			return err
		}

		// 10. 创建转账订单(status=2 成功)
		order = &model.TransferOrder{
			TransferNo:    transferNo,
			FromUserID:    fromUserID,
			FromAccountID: fromAccount.ID,
			ToUserID:      toUserID,
			ToAccountID:   toAccount.ID,
			Currency:      currency,
			Amount:        amount,
			Status:        2,
			Remark:        remark,
		}
		if err := tx.Create(order).Error; err != nil {
			// transfer_no 唯一索引冲突 → 并发重复转账,被挡住
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return ErrTransferConflict
			}
			return err
		}

		// 11. 记两条流水
		fromTxn := &model.AccountTransaction{
			TxnNo:         pkg.GenerateNo("TXN_"),
			AccountID:     from.ID,
			UserID:        from.UserID,
			BizType:       5, // 5 = 转出
			BizOrderNo:    order.TransferNo,
			Direction:     2, // 支出
			Amount:        amount,
			BalanceBefore: fromBefore,
			BalanceAfter:  fromAfter,
			Currency:      currency,
		}
		if err := tx.Create(fromTxn).Error; err != nil {
			return err
		}

		toTxn := &model.AccountTransaction{
			TxnNo:         pkg.GenerateNo("TXN_"),
			AccountID:     to.ID,
			UserID:        to.UserID,
			BizType:       6, // 6 = 转入
			BizOrderNo:    order.TransferNo,
			Direction:     1, // 收入
			Amount:        amount,
			BalanceBefore: toBefore,
			BalanceAfter:  toAfter,
			Currency:      currency,
		}
		if err := tx.Create(toTxn).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return order, nil
}

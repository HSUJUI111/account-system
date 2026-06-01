package handler

import (
	"account-system/internal/service"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

type TransferRequest struct {
	TransferNo string `json:"transfer_no"` // 客户端可选传,不传则后端生成
	FromUserID uint   `json:"from_user_id" binding:"required"`
	ToUserID   uint   `json:"to_user_id" binding:"required"`
	Currency   string `json:"currency" binding:"required"`
	Amount     string `json:"amount" binding:"required"`
	Remark     string `json:"remark"`
}

func TransferHandler(c *gin.Context) {
	var req TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误: " + err.Error()})
		return
	}

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "金额格式错误"})
		return
	}

	order, err := service.Transfer(req.TransferNo, req.FromUserID, req.ToUserID, req.Currency, amount, req.Remark)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidAmount),
			errors.Is(err, service.ErrSameAccount):
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		case errors.Is(err, service.ErrAccountNotFound):
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": err.Error()})
		case errors.Is(err, service.ErrInsufficientBalance):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"code": 422, "msg": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "转账失败: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "转账成功", "data": order})
}

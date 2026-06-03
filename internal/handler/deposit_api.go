package handler

import (
	"account-system/internal/middleware"
	"account-system/internal/service"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

type DepositOrderRequest struct {
	Currency string `json:"currency"  binding:"required"`
	Amount   string `json:"amount"  binding:"required"`
}

func CreateDepositHandler(c *gin.Context) {
	var req DepositOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "参数错误: " + err.Error(),
		})
		return
	}
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "金额格式错误",
		})
		return
	}
	userID := c.GetUint(middleware.CtxUserID)
	order, err := service.CreateDepositOrder(userID, req.Currency, amount)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidAmount):
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		case errors.Is(err, service.ErrAccountNotFound):
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "创建订单失败: " + err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "充值订单创建成功",
		"data": order,
	})
}

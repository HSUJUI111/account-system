package handler

import (
	"account-system/internal/middleware"
	"account-system/internal/service"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

type AccountRequest struct {
	Currency string `json:"currency" form:"currency" binding:"required"`
}

func CreateAccountHandler(c *gin.Context) {
	// 创建账户
	var req AccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "参数错误: " + err.Error(),
		})
		return
	}
	userID := c.GetUint(middleware.CtxUserID)
	account, err := service.CreateAccount(userID, req.Currency)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAccountExists):
			c.JSON(http.StatusConflict, gin.H{
				"code": 409,
				"msg":  err.Error(),
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": 500,
				"msg":  "创建账户失败: " + err.Error(),
			})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "账户创建成功",
		"data": account,
	})
}

// 查账户
func GetAccountHandler(c *gin.Context) {
	var req AccountRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "参数错误: " + err.Error(),
		})
		return
	}
	userID := c.GetUint(middleware.CtxUserID)
	account, err := service.GetAccount(userID, req.Currency)
	if err != nil {
		if errors.Is(err, service.ErrAccountNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"code": 404,
				"msg":  "账户不存在",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "查询账户失败: " + err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "查询账户成功",
		"data": account,
	})
}

type ConfirmDepositRequest struct {
	OrderNo string `json:"order_no" binding:"required"`
}

func ConfirmDepositHandler(c *gin.Context) {
	var req ConfirmDepositRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误" + err.Error()})
		return
	}
	if err := service.ConfirmDeposit(req.OrderNo); err != nil {
		switch {
		case errors.Is(err, service.ErrOrderNotFound):
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "入账失败: " + err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "入账成功"})
}

type CreateWithDrawRequest struct {
	Currency     string `json:"currency" binding:"required"`
	Amount       string `json:"amount" binding:"required"`
	PayeeAccount string `json:"payee_account" binding:"required"`
	PayeeName    string `json:"payee_name" binding:"required"`
	PayeeBank    string `json:"payee_bank" `
}

func CreateWithdrawHandler(c *gin.Context) {
	var req CreateWithDrawRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误" + err.Error()})
		return
	}
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "金额格式错误",
		})
	}
	userID := c.GetUint(middleware.CtxUserID)
	order, err := service.CreateWithdrawOrder(
		userID, req.Currency, amount, req.PayeeAccount, req.PayeeName, req.PayeeBank,
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidAmount),
			errors.Is(err, service.ErrInvalidPayee):
			c.JSON(http.StatusBadRequest, gin.H{
				"code": 404,
				"msg":  err.Error(),
			})
		case errors.Is(err, service.ErrAccountNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"code": 404,
				"msg":  err.Error(),
			})
		case errors.Is(err, service.ErrInsufficientBalance):
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"code": 422,
				"msg":  err.Error(),
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": 500,
				"msg":  "提款失败: " + err.Error(),
			})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "提现申请成功,正常处理",
		"data": order,
	})
}

type ConfirmWithdrawRequest struct {
	OrderNo    string `json:"order_no" binding:"required"`
	Success    *bool  `json:"success" binding:"required"` // 注意:用指针,否则 false 会被 required 校验失败
	FailReason string `json:"fail_reason"`                // 失败时才传
}

func ConfirmWithdrawHandler(c *gin.Context) {
	var req ConfirmWithdrawRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误: " + err.Error()})
		return
	}

	if err := service.ConfirmWithdraw(req.OrderNo, *req.Success, req.FailReason); err != nil {
		switch {
		case errors.Is(err, service.ErrOrderNotFound):
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "处理失败: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "处理成功"})
}

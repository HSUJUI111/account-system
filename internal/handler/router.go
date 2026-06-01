package handler

import (
	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {

	r := gin.Default()

	r.POST("/api/accounts", CreateAccount) // 创建账户
	r.GET("/api/accounts", GetAccount)     // 获取账户信息
	r.POST("/api/deposit/create", CreateDepositHandler)
	r.POST("/api/deposit/confirm", ConfirmDepositHandler)
	r.POST("/api/withdraw/create", CreateWithDrawHandler)
	r.POST("/api/withdraw/confirm", ConfirmWithdrawHandler)
	r.POST("/api/transfer", TransferHandler)
	r.GET("/api/reconcile/account/:id", ReconcileAccountHandler)
	r.GET("/api/reconcile/all", ReconcileAllHandler)
	return r
}

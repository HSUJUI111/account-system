package api

import (
	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {

	r := gin.Default()

	r.POST("/api/accounts", CreateAccount) // 创建账户
	r.GET("/api/accounts", GetAccount)     // 获取账户信息
	r.POST("/api/deposit/create", CreateDepositApi)
	r.POST("/api/deposit/confirm", ConfirmDepositApi)
	r.POST("/api/withdraw/create", CreateWithDrawApi)
	r.POST("/api/withdraw/confirm", ConfirmWithdrawHandler)
	r.POST("/api/transfer", TransferApi)
	r.GET("/api/reconcile/account/:id", ReconcileAccountApi)
	r.GET("/api/reconcile/all", ReconcileAllApi)
	return r
}

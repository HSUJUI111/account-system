package handler

import (
	"account-system/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {

	r := gin.Default()

	auth := r.Group("/api/auth")
	{
		auth.POST("/register", RegisterHandler)
		auth.POST("/login", LoginHandler)
		auth.POST("/refresh", RefreshHandler)
		auth.POST("/logout", LogoutHandler) // logout 内部从 header 取 token,这里不挂 middleware
	}

	// === 内部接口(模拟支付回调和对账,生产环境用其他鉴权)===
	internal := r.Group("/api")
	{
		internal.POST("/deposit/confirm", ConfirmDepositHandler)
		internal.POST("/withdraw/confirm", ConfirmWithdrawHandler)
		internal.GET("/reconcile/account/:id", ReconcileAccountHandler)
		internal.GET("/reconcile/all", ReconcileAllHandler)
	}

	// === 业务接口(需要 JWT 鉴权)===
	api := r.Group("/api")
	api.Use(middleware.JWTAuth()) // 这一组所有接口前面都会过 middleware
	{
		api.POST("/accounts/create", CreateAccountHandler)
		api.GET("/accounts", GetAccountHandler)
		api.POST("/deposit/create", CreateDepositHandler)
		api.POST("/withdraw/create", CreateWithdrawHandler)
		api.POST("/transfer", TransferHandler)
	}
	return r
}

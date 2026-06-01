package api

import (
	"account-system/internal/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ReconcileSingleResponse struct {
	Code int                      `json:"code"`
	Msg  string                   `json:"msg"`
	Data *service.ReconcileResult `json:"data"`
}

func ReconcileAccountApi(c *gin.Context) {
	accountIDStr := c.Param("id")
	accountID, err := strconv.ParseUint(accountIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "账户ID格式错误"})
		return
	}

	res, err := service.ReconcileAccount(uint(accountID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "对账失败: " + err.Error()})
		return
	}

	msg := "账目一致"
	if !res.IsConsistent {
		msg = "发现不一致,已写入告警表"
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": msg, "data": res})
}

func ReconcileAllApi(c *gin.Context) {
	inconsistent, err := service.ReconcileAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "对账失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":               200,
		"msg":                "全量对账完成",
		"inconsistent_count": len(inconsistent),
		"data":               inconsistent,
	})
}

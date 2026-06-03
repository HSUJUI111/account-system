package handler

import (
	"account-system/internal/service"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// ============ 请求结构 ============

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6,max=64"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// ============ Handler ============

func RegisterHandler(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误: " + err.Error()})
		return
	}

	user, err := service.Register(req.Username, req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserExists):
			c.JSON(http.StatusConflict, gin.H{"code": 409, "msg": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "注册失败: " + err.Error()})
		}
		return
	}

	// 不要在响应里返回 password(即使是 hash 也不该泄露)
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "注册成功",
		"data": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
		},
	})
}

func LoginHandler(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误: " + err.Error()})
		return
	}

	accessToken, refreshToken, user, err := service.Login(req.Username, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidLogin):
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "登录失败: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "登录成功",
		"data": gin.H{
			"user": gin.H{
				"id":       user.ID,
				"username": user.Username,
				"email":    user.Email,
			},
			"access_token":  accessToken,
			"refresh_token": refreshToken,
		},
	})
}

func RefreshHandler(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误: " + err.Error()})
		return
	}

	newAccessToken, err := service.RefreshAccessToken(req.RefreshToken)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTokenExpired):
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "refresh token 已过期,请重新登录"})
		case errors.Is(err, service.ErrInvalidToken), errors.Is(err, service.ErrTokenRevoked):
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "refresh token 无效"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "刷新失败: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "刷新成功",
		"data": gin.H{
			"access_token": newAccessToken,
		},
	})
}

func LogoutHandler(c *gin.Context) {
	// 从 Authorization header 拿 token
	token := extractToken(c)
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "缺少 Authorization header"})
		return
	}

	if err := service.Logout(token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "登出失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "登出成功"})
}

// extractToken 从 Authorization header 解析 Bearer token
// 格式:Authorization: Bearer <token>
func extractToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if auth == "" {
		return ""
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}
	return parts[1]
}

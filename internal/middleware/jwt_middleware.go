package middleware

import (
	"account-system/internal/service"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// 上下文 key,统一管理避免拼写错误
const (
	CtxUserID = "user_id"
)

// JWTAuth 是鉴权中间件:校验 Bearer token,把 user_id 塞进 context
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 从 Authorization header 取 token
		token := extractBearerToken(c)
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "缺少认证 token"})
			c.Abort() // 终止后续 handler
			return
		}

		// 2. 校验 token(内部会查 Redis 黑名单)
		claims, err := service.ParseToken(token)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrTokenExpired):
				c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "token 已过期,请重新登录"})
			case errors.Is(err, service.ErrTokenRevoked):
				c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "token 已被吊销"})
			default:
				c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "token 无效"})
			}
			c.Abort()
			return
		}

		// 3. 拒绝 refresh token 访问业务接口
		// 只有 access token 才能用来调业务接口,refresh token 只能调 /api/auth/refresh
		if claims.Type != "access" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "请使用 access token"})
			c.Abort()
			return
		}

		// 4. 把 user_id 塞进 context,handler 用 c.GetUint(CtxUserID) 取
		c.Set(CtxUserID, claims.UserID)
		c.Next()
	}
}

// extractBearerToken 从 Authorization header 解析 Bearer token
func extractBearerToken(c *gin.Context) string {
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

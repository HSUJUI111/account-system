package service

import (
	"account-system/config"
	"account-system/internal/model"
	"account-system/internal/repository"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// ============ 全局配置(在 InitAuth 时注入)============
var (
	jwtSecret       []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
)

// InitAuth 在 main 里调用一次,把 JWT 配置注入到 service 层
func InitAuth(cfg config.JWTConfig) {
	jwtSecret = []byte(cfg.Secret)
	accessTokenTTL = time.Duration(cfg.AccessTokenTTLMinutes) * time.Minute
	refreshTokenTTL = time.Duration(cfg.RefreshTokenTTLHours) * time.Hour
}

// ============ JWT Claims 结构 ============

// Claims 是 JWT payload 的结构,包含我们自定义的字段 + 标准字段
type Claims struct {
	UserID uint   `json:"user_id"`
	Type   string `json:"type"` // "access" 或 "refresh"
	jwt.RegisteredClaims
}

// ============ 密码处理 ============

// HashPassword 用 bcrypt 加密密码。bcrypt 内置随机 salt,同一密码每次哈希结果都不同
func HashPassword(plain string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword 比对明文和 hash。bcrypt 会自动从 hash 里取 salt 做比对
func CheckPassword(plain, hashed string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain))
	return err == nil
}

// ============ JWT 签发 ============

// GenerateTokens 同时生成 access 和 refresh token
func GenerateTokens(userID uint) (accessToken, refreshToken string, err error) {
	accessToken, err = generateToken(userID, "access", accessTokenTTL)
	if err != nil {
		return "", "", err
	}
	refreshToken, err = generateToken(userID, "refresh", refreshTokenTTL)
	if err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}

// generateToken 是内部辅助函数,生成单个 token
func generateToken(userID uint, tokenType string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Type:   tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(), // jti,唯一 ID,用于黑名单
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ============ JWT 校验 ============

var (
	ErrInvalidToken = errors.New("token 无效")
	ErrTokenExpired = errors.New("token 已过期")
	ErrTokenRevoked = errors.New("token 已被吊销") // 黑名单命中
)

// ParseToken 校验 token,返回 claims
func ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		// 确认签名算法是 HS256,防止算法替换攻击
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// 查 Redis 黑名单
	revoked, err := isTokenRevoked(claims.ID)
	if err != nil {
		// Redis 出错不应该让所有人登录失败。这里有两种选择:
		// A. 严格:Redis 出错就拒绝(安全优先)
		// B. 放过:Redis 出错放过 token(可用性优先)
		// 学习版我们选 A,记到 README 里说明
		return nil, fmt.Errorf("黑名单查询失败: %w", err)
	}
	if revoked {
		return nil, ErrTokenRevoked
	}

	return claims, nil
}

// ============ Redis 黑名单 ============

const blacklistKeyPrefix = "jwt:blacklist:"

// RevokeToken 把 token 加入黑名单(登出时调用)
// 传入 jti(token ID)和 token 的过期时间——TTL 设为剩余有效期,过期自动清理
func RevokeToken(jti string, expiresAt time.Time) error {
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		// token 已经过期,无需加黑名单(Redis 也会拒绝 TTL<=0 的写入)
		return nil
	}
	ctx := context.Background()
	return repository.RDB.Set(ctx, blacklistKeyPrefix+jti, "1", ttl).Err()
}

// isTokenRevoked 检查 token 是否在黑名单
func isTokenRevoked(jti string) (bool, error) {
	ctx := context.Background()
	n, err := repository.RDB.Exists(ctx, blacklistKeyPrefix+jti).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// ============ 用户业务 ============

var (
	ErrUserExists   = errors.New("用户名或邮箱已被注册")
	ErrUserNotFound = errors.New("用户不存在")
	ErrInvalidLogin = errors.New("用户名或密码错误")
)

// Register 注册新用户
func Register(username, email, password string) (*model.User, error) {
	if username == "" || email == "" || password == "" {
		return nil, errors.New("用户名、邮箱、密码不能为空")
	}

	hashed, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Username: username,
		Email:    email,
		Password: hashed,
		Status:   1,
	}

	if err := repository.DB.Create(user).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, ErrUserExists
		}
		return nil, err
	}
	return user, nil
}

// Login 用户名密码登录,返回 access + refresh token
func Login(username, password string) (accessToken, refreshToken string, user *model.User, err error) {
	var u model.User
	err = repository.DB.Where("username = ?", username).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", "", nil, ErrInvalidLogin // 故意混淆:不告诉攻击者是"用户不存在"还是"密码错"
	}
	if err != nil {
		return "", "", nil, err
	}

	if !CheckPassword(password, u.Password) {
		return "", "", nil, ErrInvalidLogin
	}

	accessToken, refreshToken, err = GenerateTokens(u.ID)
	if err != nil {
		return "", "", nil, err
	}
	return accessToken, refreshToken, &u, nil
}

// RefreshAccessToken 用 refresh token 换新的 access token
func RefreshAccessToken(refreshToken string) (string, error) {
	claims, err := ParseToken(refreshToken)
	if err != nil {
		return "", err
	}
	// 必须是 refresh 类型,access token 不能用来 refresh
	if claims.Type != "refresh" {
		return "", ErrInvalidToken
	}
	return generateToken(claims.UserID, "access", accessTokenTTL)
}

// Logout 把 access token 拉黑
func Logout(accessToken string) error {
	claims, err := ParseToken(accessToken)
	if err != nil {
		// 已经无效的 token,无需再拉黑
		return nil
	}
	return RevokeToken(claims.ID, claims.ExpiresAt.Time)
}

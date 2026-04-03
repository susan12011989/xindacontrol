package token_manager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"server/pkg/dbs"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

var (
	defaultSecret = []byte("admin.secRet#1340")
	defaultExpire = 24 * time.Hour // 默认1天
)

var def *TokenInstance

const (
	tokenKeyPrefix    = "control-server:auth_token:"
	userTokenKeyPrefx = "control-server:user_token:"
)

var (
	ErrTokenExpired  = errors.New("token expired")
	ErrTokenRevoked  = errors.New("token revoked")
	ErrTokenNotFound = errors.New("token not found")
	ErrTokenInvalid  = errors.New("token invalid")
)

func Init() {
	def = NewTokenInstance()
}

func NewTokenInstance() *TokenInstance {
	return &TokenInstance{
		secret:        defaultSecret,
		defaultExpire: defaultExpire,
	}
}

type TokenInstance struct {
	secret        []byte
	defaultExpire time.Duration
}

type TokenInfo struct {
	TokenID   string    `json:"token_id"`
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	IP        string    `json:"ip"`
	Device    string    `json:"device"`
	ExpiresAt time.Time `json:"expires_at"`
	Prefix    string    `json:"prefix"`
	TwoFA     bool      `json:"two_fa"` // 是否启用了2FA
}

func GenerateToken(userID int, username, role, ip, device string, twoFA bool) (string, error) {
	return def.GenerateToken(userID, username, role, ip, device, twoFA)
}
func BuildTokenID(userID int, uuid string) string {
	return fmt.Sprintf("%d:%s", userID, uuid)
}
func (tm *TokenInstance) GenerateToken(userID int, username, role, ip, device string, twoFA bool) (string, error) {
	// tokenID := fmt.Sprintf("%s:%d:%s", accountPrefix, userID, strings.ReplaceAll(uuid.New().String(), "-", ""))
	tokenID := BuildTokenID(userID, strings.ReplaceAll(uuid.New().String(), "-", ""))
	expiresAt := time.Now().Add(tm.defaultExpire)

	// 生成JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"jti": tokenID,
	})

	signedToken, err := token.SignedString(tm.secret)
	if err != nil {
		return "", fmt.Errorf("generate token failed: %w", err)
	}

	// 保存到Redis
	info := &TokenInfo{
		TokenID:   tokenID,
		UserID:    int64(userID),
		Username:  username,
		Role:      role,
		IP:        ip,
		Device:    device,
		ExpiresAt: expiresAt,
		Prefix:    "",
		TwoFA:     twoFA,
	}
	b, err := json.Marshal(info)
	if err != nil {
		return "", fmt.Errorf("marshal token info failed: %w", err)
	}
	ctx := context.Background()
	if err := dbs.Rds().Set(ctx, tokenRedisKey(tokenID), b, tm.defaultExpire).Err(); err != nil {
		return "", fmt.Errorf("save token to redis failed: %w", err)
	}
	// 单点登录：将用户指针指向当前token，并清理旧token（若存在）
	oldID, _ := dbs.Rds().Get(ctx, userTokenRedisKey(int64(userID))).Result()
	if oldID != "" && oldID != tokenID {
		_ = dbs.Rds().Del(ctx, tokenRedisKey(oldID)).Err()
	}
	if err := dbs.Rds().Set(ctx, userTokenRedisKey(int64(userID)), tokenID, tm.defaultExpire).Err(); err != nil {
		// 清理已写入的token值，防止残留
		_ = dbs.Rds().Del(ctx, tokenRedisKey(tokenID)).Err()
		return "", fmt.Errorf("set user token pointer failed: %w", err)
	}
	logx.Infof("uid=%d token=%s 2fa=%v", userID, signedToken, twoFA)
	return signedToken, nil
}
func ValidateToken(tokenString string) (*TokenInfo, error) {
	return def.ValidateToken(tokenString)
}
func (tm *TokenInstance) ValidateToken(tokenString string) (*TokenInfo, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return tm.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token failed: %w", err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	jtiVal, ok := claims["jti"]
	if !ok {
		return nil, fmt.Errorf("missing jti in token claims")
	}
	tokenID, ok := jtiVal.(string)
	if !ok {
		return nil, fmt.Errorf("invalid jti type")
	}
	return tm.getTokenInfo(tokenID)
}
func RefreshToken(tokenID string) error {
	return def.RefreshToken(tokenID)
}
func (tm *TokenInstance) RefreshToken(tokenID string) error {
	ctx := context.Background()
	// 读取并更新过期时间
	val, err := dbs.Rds().Get(ctx, tokenRedisKey(tokenID)).Bytes()
	if err != nil {
		if err == dbs.ErrNil {
			return fmt.Errorf("token not found")
		}
		return fmt.Errorf("get token from redis failed: %w", err)
	}
	var info TokenInfo
	if err := json.Unmarshal(val, &info); err != nil {
		return fmt.Errorf("unmarshal token info failed: %w", err)
	}
	info.ExpiresAt = time.Now().Add(tm.defaultExpire)
	b, err := json.Marshal(&info)
	if err != nil {
		return fmt.Errorf("marshal token info failed: %w", err)
	}
	if err := dbs.Rds().Set(ctx, tokenRedisKey(tokenID), b, tm.defaultExpire).Err(); err != nil {
		return fmt.Errorf("save token to redis failed: %w", err)
	}
	// 同步续期用户指针TTL（若当前指针指向该token）
	cur, _ := dbs.Rds().Get(ctx, userTokenRedisKey(info.UserID)).Result()
	if cur == tokenID {
		_ = dbs.Rds().Expire(ctx, userTokenRedisKey(info.UserID), tm.defaultExpire).Err()
	}
	return nil
}
func RevokeToken(tokenID string) error {
	return def.RevokeToken(tokenID)
}

// 兼容接口占位：不再支持 clientType/typ 的精细撤销

func (tm *TokenInstance) RevokeToken(tokenID string) error {
	ctx := context.Background()
	// 从 tokenID 中解析 userID，避免额外的读取
	uid, err := parseUserIDFromTokenID(tokenID)
	if err != nil {
		// 回退：如果格式异常，尝试读取以获取 userID
		if info, e2 := tm.getTokenInfo(tokenID); e2 == nil {
			uid = info.UserID
		} else {
			// 直接尝试删除 token 键
			_ = dbs.Rds().Del(ctx, tokenRedisKey(tokenID)).Err()
			return nil
		}
	}
	// 删除token并清理用户指针（若匹配）
	if err := dbs.Rds().Del(ctx, tokenRedisKey(tokenID)).Err(); err != nil && err != dbs.ErrNil {
		return fmt.Errorf("delete token key failed: %w", err)
	}
	// 若当前指针指向该token，则删除指针
	cur, _ := dbs.Rds().Get(ctx, userTokenRedisKey(uid)).Result()
	if cur == tokenID {
		_ = dbs.Rds().Del(ctx, userTokenRedisKey(uid)).Err()
	}
	return nil
}
func RevokeAllTokens(userID int) error {
	return def.RevokeAllTokens(userID)
}
func (tm *TokenInstance) RevokeAllTokens(userID int) error {
	uid := int64(userID)
	ctx := context.Background()
	// 读取当前指针token并删除
	cur, _ := dbs.Rds().Get(ctx, userTokenRedisKey(uid)).Result()
	if cur != "" {
		_ = dbs.Rds().Del(ctx, tokenRedisKey(cur)).Err()
	}
	// 删除用户指针
	_ = dbs.Rds().Del(ctx, userTokenRedisKey(uid)).Err()
	logx.Infof("deleted tokens for user_id=%d", userID)
	return nil
}
func (tm *TokenInstance) getTokenInfo(tokenID string) (*TokenInfo, error) {
	ctx := context.Background()
	val, err := dbs.Rds().Get(ctx, tokenRedisKey(tokenID)).Bytes()
	if err != nil {
		if err == dbs.ErrNil {
			return nil, fmt.Errorf("token not found")
		}
		return nil, fmt.Errorf("get token from redis failed: %w", err)
	}
	var info TokenInfo
	if err := json.Unmarshal(val, &info); err != nil {
		return nil, fmt.Errorf("unmarshal token info failed: %w", err)
	}
	if time.Now().After(info.ExpiresAt) {
		_ = tm.RevokeToken(tokenID)
		return nil, fmt.Errorf("token expired")
	}
	// 单点登录校验：tokenID 必须与用户当前指针一致
	cur, _ := dbs.Rds().Get(context.Background(), userTokenRedisKey(info.UserID)).Result()
	if cur == "" || cur != tokenID {
		return nil, fmt.Errorf("token revoked")
	}
	return &info, nil
}

// redis keys helpers
func tokenRedisKey(tokenID string) string {
	return tokenKeyPrefix + tokenID
}

func userTokenRedisKey(userID int64) string {
	return fmt.Sprintf("%s%d", userTokenKeyPrefx, userID)
}

func parseUserIDFromTokenID(tokenID string) (int64, error) {
	// tokenID format: "<uid>:<uuid>"
	idx := strings.IndexByte(tokenID, ':')
	if idx <= 0 {
		return 0, fmt.Errorf("invalid token id")
	}
	uidStr := tokenID[:idx]
	uid, err := strconv.ParseInt(uidStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid token id uid: %w", err)
	}
	return uid, nil
}

// ExtractTokenID 仅解析JWT返回其jti（不做存储校验）
func ExtractTokenID(tokenString string) (string, error) {
	return def.extractTokenID(tokenString)
}

func (tm *TokenInstance) extractTokenID(tokenString string) (string, error) {
	tok, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return tm.secret, nil
	})
	if err != nil {
		return "", fmt.Errorf("invalid token")
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok || !tok.Valid {
		return "", fmt.Errorf("invalid token")
	}
	jtiVal, ok := claims["jti"]
	if !ok {
		return "", fmt.Errorf("invalid token")
	}
	jti, ok := jtiVal.(string)
	if !ok {
		return "", fmt.Errorf("invalid token")
	}
	return jti, nil
}

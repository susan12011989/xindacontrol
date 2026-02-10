package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"server/internal/dbhelper"
	"server/internal/server/model"
	"server/pkg/dbs"
	"server/pkg/token_manager"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/crypto/bcrypt"
)

const (
	LoginBanTime  = 24 * time.Hour // 登录禁止时间
	MaxLoginFails = 8              // 最大登录失败次数
	failPrefix    = "admin_login_fail:%s"
)

// 内存中的挑战数据与密钥，仅进程内有效
var (
	rsaOnce        sync.Once
	rsaPrivKey     *rsa.PrivateKey
	rsaPubPEM      string
	nonceStore     = make(map[string]int64) // nonce -> expiresAt
	nonceStoreLock sync.Mutex
)

const (
	challengeTTLSeconds  = 120 // nonce 有效期（秒）
	loginRequestTimeSkew = 120 // 允许的时间窗（秒）
	rsaKeyBits           = 2048
)

func initRSA() {
	rsaOnce.Do(func() {
		key, err := rsa.GenerateKey(rand.Reader, rsaKeyBits)
		if err != nil {
			logx.Errorf("generate rsa key err: %+v", err)
			return
		}
		rsaPrivKey = key
		pubDER, _ := x509.MarshalPKIXPublicKey(&key.PublicKey)
		block := &pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}
		rsaPubPEM = string(pem.EncodeToMemory(block))
	})
}

func Login(ip string, req model.LoginReq) (any, error) {
	return LoginWithTwoFA(ip, req.Username, req.Password, "", "")
}

// LoginWithTwoFA 带2FA的登录
func LoginWithTwoFA(ip, username, password, twoFACode, device string) (any, error) {
	// 检查是否被限制登录
	failKey := fmt.Sprintf(failPrefix, username)
	failCount, _ := dbs.Rds().Get(context.Background(), failKey).Int()
	if failCount >= MaxLoginFails {
		return nil, errors.New("登录错误次数过多，请24小时后再试")
	}

	user, err := dbhelper.GetSysUserByUsername(username)
	if err != nil {
		logx.Errorf("%s login err: %+v", username, err)
		recordLoginFail(username)
		return "", errors.New("用户名或密码错误")
	}
	// 比较密码
	if user.Password != password {
		logx.Errorf("%s login password err", username)
		recordLoginFail(username)
		return "", errors.New("用户名或密码错误")
	}

	// 检查2FA
	if user.TwoFactorEnabled == 1 {
		if twoFACode == "" {
			return nil, errors.New("需要2FA验证码")
		}
		if !VerifyTwoFACode(user.TwoFactorSecret, twoFACode) {
			logx.Errorf("%s 2FA code err", username)
			recordLoginFail(username)
			return nil, errors.New("2FA验证码错误")
		}
	}

	// 登录成功
	token_manager.RevokeAllTokens(int(user.Id)) // 单点登录
	dbs.Rds().Del(context.Background(), failKey)
	token, err := token_manager.GenerateToken(user.Id, user.Username, ip, device, user.TwoFactorEnabled == 1)
	if err != nil {
		logx.Errorf("%s login err: %+v", username, err)
		return nil, err
	}
	return map[string]any{
		"token":              token,
		"two_factor_enabled": user.TwoFactorEnabled == 1,
	}, nil
}

// HashPassword 使用 bcrypt 对密码进行加密
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func recordLoginFail(username string) {
	ctx := context.Background()
	key := fmt.Sprintf(failPrefix, username)

	// 增加失败次数
	count, _ := dbs.Rds().Incr(ctx, key).Result()
	if count == 1 { // 第一次失败，设置过期时间
		dbs.Rds().Expire(ctx, key, LoginBanTime)
	}
}

// GetChallenge 返回一次性 nonce 与 RSA 公钥
func GetChallenge() (model.ChallengeResp, error) {
	initRSA()
	if rsaPrivKey == nil || rsaPubPEM == "" {
		return model.ChallengeResp{}, errors.New("初始化密钥失败")
	}
	// 生成随机 nonce（base64）
	nonceBytes := make([]byte, 24)
	if _, err := rand.Read(nonceBytes); err != nil {
		return model.ChallengeResp{}, err
	}
	nonce := base64.StdEncoding.EncodeToString(nonceBytes)
	expiresAt := time.Now().Add(time.Duration(challengeTTLSeconds) * time.Second).Unix()
	// 存储 nonce
	nonceStoreLock.Lock()
	nonceStore[nonce] = expiresAt
	nonceStoreLock.Unlock()
	return model.ChallengeResp{Nonce: nonce, PubPEM: rsaPubPEM, Expires: expiresAt}, nil
}

// LoginEncrypted 处理前端 RSA-OAEP 加密后的登录
func LoginEncrypted(ip string, req model.EncryptedLoginReq) (any, error) {
	initRSA()
	if rsaPrivKey == nil {
		return nil, errors.New("密钥未初始化")
	}
	// Base64 解码
	cipherBytes, err := base64.StdEncoding.DecodeString(req.Cipher)
	if err != nil {
		return nil, errors.New("密文格式错误")
	}
	// RSA-OAEP 解密（SHA-256）
	label := []byte("")
	plain, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, rsaPrivKey, cipherBytes, label)
	if err != nil {
		return nil, errors.New("解密失败")
	}
	// 解析负载
	var payload model.EncryptedLoginPayload
	if err := json.Unmarshal(plain, &payload); err != nil {
		return nil, errors.New("明文格式错误")
	}
	// 校验时间窗
	now := time.Now().Unix()
	if payload.Ts <= 0 || abs64(now-payload.Ts) > loginRequestTimeSkew {
		return nil, errors.New("请求已过期")
	}
	// 校验 nonce 一次性与有效期
	nonceStoreLock.Lock()
	expiresAt, ok := nonceStore[payload.Nonce]
	if ok {
		delete(nonceStore, payload.Nonce) // 一次性
	}
	nonceStoreLock.Unlock()
	if !ok || expiresAt < now {
		return nil, errors.New("无效的 nonce")
	}
	// 走常规登录（使用明文用户名、密码和2FA验证码）
	return LoginWithTwoFA(ip, payload.Username, payload.Password, payload.TwoFACode, "")
}

// LoginEncryptedWithUA 透传 UA 并登录
func LoginEncryptedWithUA(ip, ua string, req model.EncryptedLoginReq) (any, error) {
	initRSA()
	if rsaPrivKey == nil {
		return nil, errors.New("密钥未初始化")
	}
	cipherBytes, err := base64.StdEncoding.DecodeString(req.Cipher)
	if err != nil {
		return nil, errors.New("密文格式错误")
	}
	label := []byte("")
	plain, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, rsaPrivKey, cipherBytes, label)
	if err != nil {
		return nil, errors.New("解密失败")
	}
	var payload model.EncryptedLoginPayload
	if err := json.Unmarshal(plain, &payload); err != nil {
		return nil, errors.New("明文格式错误")
	}
	now := time.Now().Unix()
	if payload.Ts <= 0 || abs64(now-payload.Ts) > loginRequestTimeSkew {
		return nil, errors.New("请求已过期")
	}
	nonceStoreLock.Lock()
	expiresAt, ok := nonceStore[payload.Nonce]
	if ok {
		delete(nonceStore, payload.Nonce)
	}
	nonceStoreLock.Unlock()
	if !ok || expiresAt < now {
		return nil, errors.New("无效的 nonce")
	}
	return LoginWithTwoFA(ip, payload.Username, payload.Password, payload.TwoFACode, ua)
}

func abs64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

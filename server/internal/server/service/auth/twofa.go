package auth

import (
	"errors"
	"fmt"
	"server/internal/dbhelper"
	"server/internal/server/model"
	"server/pkg/dbs"

	"github.com/pquerna/otp/totp"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	issuer = "TeamGram Admin" // TOTP发行者名称
)

// GenerateTwoFASecret 生成新的2FA密钥
func GenerateTwoFASecret(username string) (model.TwoFASetupResp, error) {
	// 生成TOTP密钥
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: username,
	})
	if err != nil {
		logx.Errorf("generate totp key error: %+v", err)
		return model.TwoFASetupResp{}, errors.New("生成2FA密钥失败")
	}

	// 返回密钥和二维码URL
	return model.TwoFASetupResp{
		Secret:  key.Secret(),
		QRCode:  key.URL(),
		Enabled: false,
	}, nil
}

// VerifyTwoFACode 验证TOTP验证码
func VerifyTwoFACode(secret, code string) bool {
	if secret == "" || code == "" {
		return false
	}
	return totp.Validate(code, secret)
}

// EnableTwoFA 启用2FA
func EnableTwoFA(username, code string) error {
	// 获取用户
	user, err := dbhelper.GetSysUserByUsername(username)
	if err != nil {
		return err
	}

	// 如果已经启用，返回错误
	if user.TwoFactorEnabled == 1 {
		return errors.New("2FA已启用")
	}

	// 检查是否已有密钥
	if user.TwoFactorSecret == "" {
		return errors.New("请先获取2FA设置信息")
	}

	// 使用数据库中已保存的密钥验证验证码
	if !VerifyTwoFACode(user.TwoFactorSecret, code) {
		return errors.New("验证码错误")
	}

	// 启用2FA
	user.TwoFactorEnabled = 1

	_, err = dbs.DBAdmin.ID(user.Id).Cols("two_factor_enabled").Update(user)
	if err != nil {
		logx.Errorf("enable two factor failed: %+v", err)
		return errors.New("启用2FA失败")
	}

	logx.Infof("user %s enabled 2FA", username)
	return nil
}

// DisableTwoFA 禁用2FA
func DisableTwoFA(username, password string) error {
	// 获取用户
	user, err := dbhelper.GetSysUserByUsername(username)
	if err != nil {
		return err
	}

	// 验证密码
	if user.Password != password {
		return errors.New("密码错误")
	}

	// 如果未启用，返回错误
	if user.TwoFactorEnabled == 0 {
		return errors.New("2FA未启用")
	}

	// 清除2FA设置
	user.TwoFactorSecret = ""
	user.TwoFactorEnabled = 0

	_, err = dbs.DBAdmin.ID(user.Id).Cols("two_factor_secret", "two_factor_enabled").Update(user)
	if err != nil {
		logx.Errorf("disable two factor failed: %+v", err)
		return errors.New("禁用2FA失败")
	}

	logx.Infof("user %s disabled 2FA", username)
	return nil
}

// GetTwoFAStatus 获取2FA状态
func GetTwoFAStatus(username string) (model.TwoFAStatusResp, error) {
	user, err := dbhelper.GetSysUserByUsername(username)
	if err != nil {
		return model.TwoFAStatusResp{}, err
	}

	return model.TwoFAStatusResp{
		Enabled: user.TwoFactorEnabled == 1,
	}, nil
}

// GetTwoFASetupInfo 获取2FA设置信息（用于显示二维码）
func GetTwoFASetupInfo(username string) (model.TwoFASetupResp, error) {
	user, err := dbhelper.GetSysUserByUsername(username)
	if err != nil {
		return model.TwoFASetupResp{}, err
	}

	// 如果已启用，返回当前配置
	if user.TwoFactorEnabled == 1 && user.TwoFactorSecret != "" {
		qrURL := fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s",
			issuer, username, user.TwoFactorSecret, issuer)
		return model.TwoFASetupResp{
			Secret:  user.TwoFactorSecret,
			QRCode:  qrURL,
			Enabled: true,
		}, nil
	}

	// 如果未启用但已有密钥（准备启用中），返回现有密钥
	if user.TwoFactorSecret != "" {
		qrURL := fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s",
			issuer, username, user.TwoFactorSecret, issuer)
		return model.TwoFASetupResp{
			Secret:  user.TwoFactorSecret,
			QRCode:  qrURL,
			Enabled: false,
		}, nil
	}

	// 否则生成新的密钥并保存到数据库（但不启用）
	setupResp, err := GenerateTwoFASecret(username)
	if err != nil {
		return model.TwoFASetupResp{}, err
	}

	// 保存密钥到数据库（enabled保持为0）
	user.TwoFactorSecret = setupResp.Secret
	_, err = dbs.DBAdmin.ID(user.Id).Cols("two_factor_secret").Update(user)
	if err != nil {
		logx.Errorf("save two factor secret failed: %+v", err)
		return model.TwoFASetupResp{}, errors.New("保存2FA密钥失败")
	}

	return setupResp, nil
}

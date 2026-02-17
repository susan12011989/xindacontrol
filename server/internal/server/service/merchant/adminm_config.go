package merchant

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"server/internal/dbhelper"
	"server/internal/server/cfg"
	"server/pkg/entity"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// getMerchantAPIURL 获取商户API地址
func getMerchantAPIURL(merchantNo string, path string) (string, error) {
	merchant, err := dbhelper.GetMerchantByNo(merchantNo)
	if err != nil {
		return "", err
	}
	if merchant.ServerIP == "" {
		return "", fmt.Errorf("商户服务器IP为空")
	}
	// 使用商户配置的端口，默认 8084（商户后台管理端口）
	port := merchant.Port
	if port == 0 {
		port = 8084
	}
	return fmt.Sprintf("http://%s:%d%s", merchant.ServerIP, port, path), nil
}

// doMerchantRequest 向商户发起HTTP请求
func doMerchantRequest(method, url string, body interface{}) (*http.Response, error) {
	if cfg.C.MerchantAPI == nil {
		return nil, fmt.Errorf("MerchantAPI配置未设置")
	}

	var reqBody *bytes.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(jsonData)
	} else {
		reqBody = bytes.NewReader(nil)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(cfg.C.MerchantAPI.Username, cfg.C.MerchantAPI.Password)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	return client.Do(req)
}

// ============ Sensitive Contents ============

// SaveAdminmSensitiveContents 保存敏感词配置到商户
func SaveAdminmSensitiveContents(merchantNo string, contents []*entity.SensitiveContent) error {
	url, err := getMerchantAPIURL(merchantNo, "/v1/control/config/sensitive_contents")
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"contents": contents,
	}

	resp, err := doMerchantRequest("POST", url, payload)
	if err != nil {
		logx.Errorf("保存商户敏感词配置失败: merchant=%s, err=%v", merchantNo, err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("保存失败，状态码: %d", resp.StatusCode)
	}

	logx.Infof("敏感词配置已保存: merchant=%s, count=%d", merchantNo, len(contents))
	return nil
}

// ============ SMS Config ============

// GetAdminmSmsConfig 从商户服务器读取短信配置
func GetAdminmSmsConfig(merchantNo string) (*entity.SmsConfig, error) {
	url, err := getMerchantAPIURL(merchantNo, "/v1/control/config/sms")
	if err != nil {
		return nil, err
	}

	resp, err := doMerchantRequest("GET", url, nil)
	if err != nil {
		logx.Errorf("获取商户短信配置失败: merchant=%s, err=%v", merchantNo, err)
		return nil, fmt.Errorf("请求商户服务器失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("获取失败，状态码: %d", resp.StatusCode)
	}

	var result struct {
		Config *entity.SmsConfig `json:"config"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	return result.Config, nil
}

// SaveAdminmSmsConfig 保存短信配置到商户
func SaveAdminmSmsConfig(merchantNo string, config *entity.SmsConfig) error {
	url, err := getMerchantAPIURL(merchantNo, "/v1/control/config/sms")
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"config": config,
	}

	resp, err := doMerchantRequest("POST", url, payload)
	if err != nil {
		logx.Errorf("保存商户短信配置失败: merchant=%s, err=%v", merchantNo, err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("保存失败，状态码: %d", resp.StatusCode)
	}

	logx.Infof("短信配置已保存: merchant=%s", merchantNo)
	return nil
}

// ============ System User Nickname ============

// SaveAdminmSystemNickname 保存系统用户昵称到商户
func SaveAdminmSystemNickname(merchantNo string, firstName string) error {
	url, err := getMerchantAPIURL(merchantNo, "/v1/control/config/system_user_nickname")
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"first_name": firstName,
	}

	resp, err := doMerchantRequest("POST", url, payload)
	if err != nil {
		logx.Errorf("保存商户系统昵称失败: merchant=%s, err=%v", merchantNo, err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("保存失败，状态码: %d", resp.StatusCode)
	}

	logx.Infof("系统昵称已保存: merchant=%s, firstName=%s", merchantNo, firstName)
	return nil
}

// ============ Clear Data ============

// ClearMerchantData 清除商户所有用户数据（保留系统账号和配置）
func ClearMerchantData(merchantNo string) error {
	url, err := getMerchantAPIURL(merchantNo, "/v1/control/data/clear")
	if err != nil {
		return err
	}

	resp, err := doMerchantRequest("POST", url, nil)
	if err != nil {
		logx.Errorf("清除商户数据失败: merchant=%s, err=%v", merchantNo, err)
		return fmt.Errorf("请求商户服务器失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("清除失败，状态码: %d", resp.StatusCode)
	}

	logx.Infof("商户数据已清除: merchant=%s", merchantNo)
	return nil
}

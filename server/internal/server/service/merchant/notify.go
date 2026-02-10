package merchant

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"server/internal/server/cfg"
	"server/pkg/entity"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// NotifyConfigUpdate 向商户服务推送配置更新
// 推送 control 数据库中存储的商户配置
func NotifyConfigUpdate(merchant *entity.Merchants) error {
	if merchant.ServerIP == "" {
		return fmt.Errorf("商户服务器IP为空")
	}

	if cfg.C.MerchantAPI == nil {
		logx.Errorf("MerchantAPI配置未设置，无法推送配置更新")
		return fmt.Errorf("MerchantAPI配置未设置")
	}

	// 构建商户服务的回调地址（默认端口10002）
	url := fmt.Sprintf("http://%s:10002/v1/control/config/update", merchant.ServerIP)

	// 推送 control 数据库中存储的配置
	payload := map[string]interface{}{
		"merchant_no":           merchant.No,
		"name":                  merchant.Name,
		"status":                merchant.Status,
		"expired_at":            merchant.ExpiredAt.Unix(),
		"package_configuration": merchant.PackageConfiguration,
		"configs":               merchant.Configs,
		"app_configs":           merchant.AppConfigs,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化payload失败: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.SetBasicAuth(cfg.C.MerchantAPI.Username, cfg.C.MerchantAPI.Password)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logx.Errorf("推送配置更新失败: merchant=%s, url=%s, err=%v", merchant.No, url, err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		logx.Errorf("推送配置更新失败: merchant=%s, status=%d", merchant.No, resp.StatusCode)
		return fmt.Errorf("推送失败，状态码: %d", resp.StatusCode)
	}

	logx.Infof("配置更新已推送: merchant=%s, url=%s", merchant.No, url)
	return nil
}

// AsyncNotifyConfigUpdate 异步推送配置更新
func AsyncNotifyConfigUpdate(merchant *entity.Merchants) {
	go func() {
		if err := NotifyConfigUpdate(merchant); err != nil {
			logx.Errorf("异步推送配置更新失败: merchant=%s, err=%v", merchant.No, err)
		}
	}()
}

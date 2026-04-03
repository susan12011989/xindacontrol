package merchant

import (
	"encoding/json"
	"fmt"
	"server/pkg/entity"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// QueryAppLogs 查询商户应用日志列表
func QueryAppLogs(merchantNo string, page, size int, keyword string) (*entity.AppLogQueryResp, error) {
	// 构建查询参数
	path := fmt.Sprintf("/v1/control/app_logs?page=%d&size=%d", page, size)
	if keyword != "" {
		path += fmt.Sprintf("&keyword=%s", keyword)
	}

	url, err := getMerchantAPIURL(merchantNo, path)
	if err != nil {
		return &entity.AppLogQueryResp{Err: err.Error()}, nil
	}

	resp, err := doMerchantRequestWithTimeout("GET", url, nil, 60*time.Second)
	if err != nil {
		logx.Errorf("查询商户应用日志失败: merchant=%s, err=%v", merchantNo, err)
		return &entity.AppLogQueryResp{Err: fmt.Sprintf("请求失败: %v", err)}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return &entity.AppLogQueryResp{Err: fmt.Sprintf("请求失败，状态码: %d", resp.StatusCode)}, nil
	}

	var result entity.AppLogQueryResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return &entity.AppLogQueryResp{Err: fmt.Sprintf("解析响应失败: %v", err)}, nil
	}

	return &result, nil
}

package merchant

import (
	"fmt"
	"server/pkg/entity"

	"github.com/zeromicro/go-zero/core/logx"
)

// SendAnnouncement 向商户发送系统公告
func SendAnnouncement(merchantNo string, req *entity.AnnouncementReq) error {
	url, err := getMerchantAPIURL(merchantNo, "/v1/control/announcement")
	if err != nil {
		return err
	}

	resp, err := doMerchantRequest("POST", url, req)
	if err != nil {
		logx.Errorf("发送商户公告失败: merchant=%s, err=%v", merchantNo, err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("发送失败，状态码: %d", resp.StatusCode)
	}

	logx.Infof("公告已发送: merchant=%s, text=%s", merchantNo, truncateText(req.Text, 50))
	return nil
}

// truncateText 截断文本用于日志输出
func truncateText(text string, maxLen int) string {
	runes := []rune(text)
	if len(runes) <= maxLen {
		return text
	}
	return string(runes[:maxLen]) + "..."
}

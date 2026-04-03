package deploy

import (
	"fmt"
	"server/internal/server/model"

	"github.com/zeromicro/go-zero/core/logx"
)

// ConfigureAMIAppNode SSH 到 App 节点，使用 DeployNode 完整重新生成配置
// AMI 的优势仅在于 Docker 镜像和二进制已预装，配置文件全部重新生成以避免残留
func ConfigureAMIAppNode(serverId int, dbPrivateIP, minioPrivateIP, appPublicIP string, progressFn func(string)) error {
	if progressFn == nil {
		progressFn = func(msg string) { logx.Infof("[AMIPostConfig] %s", msg) }
	}

	progressFn("使用 DeployNode 重新生成完整配置...")

	// 构建 DeployNode 请求，走标准部署路径
	req := model.DeployNodeReq{
		ServerId:  serverId,
		NodeRole:  "app",
		DBHost:    dbPrivateIP,
		MinioHost: minioPrivateIP,
	}

	resp, err := DeployNodeByServerId(req, "cluster_wizard_ami")
	if err != nil {
		return fmt.Errorf("DeployNode 失败: %v", err)
	}
	if !resp.Success {
		return fmt.Errorf("DeployNode 未成功: %s", resp.Message)
	}

	progressFn(fmt.Sprintf("配置完成: %s", resp.Message))
	return nil
}

package merchant

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"server/internal/dbhelper"
	"server/internal/server/model"
	"server/internal/server/utils"
	"server/pkg/dbs"
	"server/pkg/entity"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// SyncMerchantSubscribers 同步商户群成员到 WuKongIM 频道订阅者
// 从 MySQL 读取所有活跃群及成员，逐群调用 WuKongIM 的 /channel/subscriber_add 接口
func SyncMerchantSubscribers(merchantId int) (*model.SyncSubscribersResp, error) {
	merchant, err := dbhelper.GetMerchantByID(merchantId)
	if err != nil {
		return nil, fmt.Errorf("获取商户失败: %v", err)
	}

	// 获取商户服务器
	var servers []entity.Servers
	err = dbs.DBAdmin.Where("merchant_id = ? AND server_type = 1 AND status = 1", merchant.Id).Find(&servers)
	if err != nil {
		return nil, fmt.Errorf("查询商户服务器失败: %v", err)
	}
	if len(servers) == 0 {
		return nil, fmt.Errorf("未找到商户服务器")
	}

	// 判断集群/单机模式，确定 DB 节点和 App 节点
	dbServer, appServer, err := resolveDBAndAppServers(merchant.Id, servers)
	if err != nil {
		return nil, err
	}

	// 1. SSH 到 DB 节点查询群成员
	dbSSH := &utils.SSHClient{
		Host:       dbServer.Host,
		Port:       dbServer.Port,
		Username:   dbServer.Username,
		Password:   dbServer.Password,
		PrivateKey: dbServer.PrivateKey,
	}
	defer dbSSH.Close()

	if err := dbSSH.Connect(); err != nil {
		return nil, fmt.Errorf("SSH连接DB节点失败: %v", err)
	}

	// 查询所有活跃群及其成员
	mysqlCmd := `docker exec tsdd-mysql mysql -uroot -p"$MYSQL_ROOT_PASSWORD" tsdd -N -e "SELECT g.group_no, GROUP_CONCAT(gm.uid) FROM ` + "`group`" + ` g JOIN group_member gm ON g.group_no=gm.group_no WHERE g.status=1 AND gm.is_deleted=0 GROUP BY g.group_no"`
	output, err := dbSSH.ExecuteCommandWithTimeout(mysqlCmd, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("查询群成员失败: %v, output: %s", err, output)
	}

	// 2. 解析输出
	groups := parseGroupMembers(output)
	if len(groups) == 0 {
		return &model.SyncSubscribersResp{
			Message: "没有找到需要同步的群组",
		}, nil
	}

	// 3. 调用 WuKongIM API 同步订阅者
	// WuKongIM API 端口: 5002（Docker 内部端口）
	wkimBaseURL := fmt.Sprintf("http://%s:5002", appServer.Host)

	resp := &model.SyncSubscribersResp{
		TotalGroups: len(groups),
	}

	for groupNo, members := range groups {
		resp.TotalMembers += len(members)
		err := syncChannelSubscribers(wkimBaseURL, groupNo, members)
		if err != nil {
			logx.Errorf("同步群订阅者失败: group=%s, err=%v", groupNo, err)
			resp.FailedGroups++
		} else {
			resp.SyncedGroups++
		}
	}

	resp.Message = fmt.Sprintf("同步完成: %d/%d 群成功，共 %d 成员", resp.SyncedGroups, resp.TotalGroups, resp.TotalMembers)
	logx.Infof("[SyncSubscribers] merchant=%d, %s", merchantId, resp.Message)
	return resp, nil
}

// resolveDBAndAppServers 确定 DB 节点和 App 节点
func resolveDBAndAppServers(merchantId int, servers []entity.Servers) (*entity.Servers, *entity.Servers, error) {
	// 检查是否为集群模式（有 ClusterNodes 记录）
	var clusterNodes []entity.ClusterNodes
	err := dbs.DBAdmin.Where("merchant_id = ?", merchantId).Find(&clusterNodes)
	if err != nil {
		return nil, nil, fmt.Errorf("查询集群节点失败: %v", err)
	}

	if len(clusterNodes) > 0 {
		// 集群模式：按角色查找
		var dbServer, appServer *entity.Servers
		for _, node := range clusterNodes {
			for i := range servers {
				if servers[i].Id == node.ServerId {
					switch node.NodeRole {
					case entity.ClusterRoleDB:
						dbServer = &servers[i]
					case entity.ClusterRoleApp:
						if appServer == nil {
							appServer = &servers[i]
						}
					}
				}
			}
		}
		if dbServer == nil {
			return nil, nil, fmt.Errorf("未找到集群DB节点")
		}
		if appServer == nil {
			return nil, nil, fmt.Errorf("未找到集群App节点")
		}
		return dbServer, appServer, nil
	}

	// 单机模式：同一台服务器
	return &servers[0], &servers[0], nil
}

// parseGroupMembers 解析 MySQL 输出为 map[groupNo][]uid
func parseGroupMembers(output string) map[string][]string {
	groups := make(map[string][]string)
	// 过滤掉 STDERR 部分（mysql 密码警告等）
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "mysql:") || strings.HasPrefix(line, "Warning") || strings.HasPrefix(line, "STDERR:") {
			continue
		}
		// 格式: group_no\tuid1,uid2,uid3
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		groupNo := strings.TrimSpace(parts[0])
		uidsStr := strings.TrimSpace(parts[1])
		if groupNo == "" || uidsStr == "" {
			continue
		}
		uids := strings.Split(uidsStr, ",")
		validUids := make([]string, 0, len(uids))
		for _, uid := range uids {
			uid = strings.TrimSpace(uid)
			if uid != "" {
				validUids = append(validUids, uid)
			}
		}
		if len(validUids) > 0 {
			groups[groupNo] = validUids
		}
	}
	return groups
}

// syncChannelSubscribers 调用 WuKongIM API 添加频道订阅者
func syncChannelSubscribers(wkimBaseURL string, channelId string, subscribers []string) error {
	url := wkimBaseURL + "/channel/subscriber_add"

	payload := map[string]interface{}{
		"channel_id":   channelId,
		"channel_type": 2, // 群组类型
		"subscribers":  subscribers,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("请求WuKongIM失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("WuKongIM返回错误状态码: %d", resp.StatusCode)
	}

	return nil
}

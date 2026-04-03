package wukongim

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"server/internal/server/model"
	"server/pkg/dbs"
	"server/pkg/entity"
	"strings"
	"time"
)

const wukongIMPort = 5001

// getServerHost 根据 serverId 获取运行 WuKongIM 的服务器 IP
// 如果传入的 serverId 本身就是 app/all 角色节点，直接返回其 IP
// 否则通过商户关联找到同商户的 app 角色节点
func getServerHost(serverId int) (string, error) {
	// 先查是否有商户节点关联此 serverId
	var node entity.MerchantServiceNodes
	has, err := dbs.DBAdmin.Where("server_id = ?", serverId).Get(&node)
	if err != nil {
		return "", fmt.Errorf("查询服务节点失败: %v", err)
	}

	if has {
		// 如果此节点本身是 app 或 all 角色，直接用它的 host
		if node.Role == "app" || node.Role == "all" {
			return node.Host, nil
		}
		// 否则找同商户的 app 角色节点（WuKongIM 运行在 app 节点上）
		var appNode entity.MerchantServiceNodes
		appHas, err := dbs.DBAdmin.Where("merchant_id = ? AND role = 'app'", node.MerchantId).Get(&appNode)
		if err != nil {
			return "", fmt.Errorf("查询app节点失败: %v", err)
		}
		if appHas {
			return appNode.Host, nil
		}
		// 尝试 all 角色（单机部署）
		var allNode entity.MerchantServiceNodes
		allHas, err := dbs.DBAdmin.Where("merchant_id = ? AND role = 'all'", node.MerchantId).Get(&allNode)
		if err != nil {
			return "", fmt.Errorf("查询all节点失败: %v", err)
		}
		if allHas {
			return allNode.Host, nil
		}
		return "", fmt.Errorf("商户 %d 未找到运行WuKongIM的节点", node.MerchantId)
	}

	// 没有商户关联，从 servers 表查 IP，再通过 merchants.server_ip 反查商户
	var server entity.Servers
	has, err = dbs.DBAdmin.Where("id = ?", serverId).Get(&server)
	if err != nil {
		return "", fmt.Errorf("查询服务器失败: %v", err)
	}
	if !has {
		return "", fmt.Errorf("服务器不存在")
	}

	// 尝试通过 IP 匹配商户主 IP，再找 app 节点
	var merchant entity.Merchants
	has, err = dbs.DBAdmin.Where("server_ip = ?", server.Host).Get(&merchant)
	if err == nil && has {
		// 单机商户直接用 server_ip
		var appNode entity.MerchantServiceNodes
		appHas, _ := dbs.DBAdmin.Where("merchant_id = ? AND (role = 'app' OR role = 'all')", merchant.Id).Get(&appNode)
		if appHas {
			return appNode.Host, nil
		}
		return server.Host, nil
	}

	return "", fmt.Errorf("服务器 %s 未关联任何商户的WuKongIM节点", server.Host)
}

// ListWuKongIMNodes 返回所有运行 WuKongIM 的节点（app/all 角色）
func ListWuKongIMNodes() ([]map[string]interface{}, error) {
	var nodes []entity.MerchantServiceNodes
	err := dbs.DBAdmin.Where("role = 'app' OR role = 'all'").Find(&nodes)
	if err != nil {
		return nil, fmt.Errorf("查询WuKongIM节点失败: %v", err)
	}

	result := make([]map[string]interface{}, 0, len(nodes))
	for _, n := range nodes {
		// 获取商户名称
		var merchant entity.Merchants
		dbs.DBAdmin.Where("id = ?", n.MerchantId).Get(&merchant)

		// 获取关联 server ID
		sid := n.ServerId
		if sid == 0 {
			// 通过 host 反查 servers 表
			var s entity.Servers
			if has, _ := dbs.DBAdmin.Where("host = ?", n.Host).Get(&s); has {
				sid = s.Id
			}
		}

		if sid > 0 {
			result = append(result, map[string]interface{}{
				"server_id":    sid,
				"host":         n.Host,
				"merchant_no":  merchant.No,
				"merchant_name": merchant.Name,
			})
		}
	}
	return result, nil
}

// buildURL 构建 WuKongIM API URL
func buildURL(host, path string) string {
	return fmt.Sprintf("http://%s:%d%s", host, wukongIMPort, path)
}

// doGet 发起 GET 请求到 WuKongIM
func doGet(url string) ([]byte, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("请求WuKongIM失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("WuKongIM返回错误(HTTP %d): %s", resp.StatusCode, string(body))
	}
	return body, nil
}

// doPost 发起 POST 请求到 WuKongIM
func doPost(url string, data any) ([]byte, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("序列化请求数据失败: %v", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, fmt.Errorf("请求WuKongIM失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("WuKongIM返回错误(HTTP %d): %s", resp.StatusCode, string(body))
	}
	return body, nil
}

// GetVarz 获取系统变量信息
func GetVarz(req model.WuKongIMBaseReq) (*model.WuKongIMVarzResp, error) {
	host, err := getServerHost(req.ServerId)
	if err != nil {
		return nil, err
	}

	body, err := doGet(buildURL(host, "/varz"))
	if err != nil {
		return nil, err
	}

	var resp model.WuKongIMVarzResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}
	return &resp, nil
}

// GetConnz 获取连接信息
func GetConnz(req model.WuKongIMConnzReq) (*model.WuKongIMConnzResp, error) {
	host, err := getServerHost(req.ServerId)
	if err != nil {
		return nil, err
	}

	// 构建查询参数
	path := fmt.Sprintf("/connz?offset=%d&limit=%d", req.Offset, req.Limit)
	if req.UID != "" {
		path += "&uid=" + req.UID
	}
	if req.Sort != "" {
		path += "&sort=" + req.Sort
	}

	body, err := doGet(buildURL(host, path))
	if err != nil {
		return nil, err
	}

	var resp model.WuKongIMConnzResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}
	return &resp, nil
}

// GetOnlineStatus 查询用户在线状态
func GetOnlineStatus(req model.WuKongIMOnlineStatusReq) ([]model.WuKongIMOnlineStatusItem, error) {
	host, err := getServerHost(req.ServerId)
	if err != nil {
		return nil, err
	}

	body, err := doPost(buildURL(host, "/user/onlinestatus"), req.UIDs)
	if err != nil {
		return nil, err
	}

	var resp []model.WuKongIMOnlineStatusItem
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}
	return resp, nil
}

// DeviceQuit 强制设备下线
func DeviceQuit(req model.WuKongIMDeviceQuitReq) error {
	host, err := getServerHost(req.ServerId)
	if err != nil {
		return err
	}

	postData := map[string]any{
		"uid":         req.UID,
		"device_flag": req.DeviceFlag,
	}
	_, err = doPost(buildURL(host, "/user/device_quit"), postData)
	return err
}

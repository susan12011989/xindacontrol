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

// getServerHost 根据 serverId 获取服务器 IP
func getServerHost(serverId int) (string, error) {
	var server entity.Servers
	has, err := dbs.DBAdmin.Where("id = ?", serverId).Get(&server)
	if err != nil {
		return "", fmt.Errorf("查询服务器失败: %v", err)
	}
	if !has {
		return "", fmt.Errorf("服务器不存在")
	}
	return server.Host, nil
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

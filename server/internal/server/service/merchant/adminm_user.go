package merchant

import (
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"server/internal/server/model"
	"server/pkg/entity"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"
)

// normalizeAllowedIPs 规范化IP白名单格式
// 1. 将中文逗号替换为英文逗号
// 2. 去除多余空格
// 3. 校验每个IP格式
func normalizeAllowedIPs(input string) (string, error) {
	if input == "" {
		return "", nil
	}

	// 替换中文逗号为英文逗号
	input = strings.ReplaceAll(input, "，", ",")
	// 替换中文分号为英文逗号
	input = strings.ReplaceAll(input, "；", ",")
	input = strings.ReplaceAll(input, ";", ",")
	// 替换换行符为逗号
	input = strings.ReplaceAll(input, "\n", ",")
	input = strings.ReplaceAll(input, "\r", "")

	// 分割并处理每个IP
	parts := strings.Split(input, ",")
	validIPs := make([]string, 0, len(parts))

	// IP地址正则（简单校验）
	ipRegex := regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)

	for _, part := range parts {
		ip := strings.TrimSpace(part)
		if ip == "" {
			continue
		}

		// 校验IP格式
		if !ipRegex.MatchString(ip) {
			return "", fmt.Errorf("无效的IP地址格式: %s", ip)
		}

		// 进一步校验IP地址有效性
		if net.ParseIP(ip) == nil {
			return "", fmt.Errorf("无效的IP地址: %s", ip)
		}

		validIPs = append(validIPs, ip)
	}

	return strings.Join(validIPs, ","), nil
}

// CreateAdminmUser 创建商户管理用户
func CreateAdminmUser(req *model.CreateAdminmUserReq) error {
	url, err := getMerchantAPIURL(req.MerchantNo, "/v1/control/adminm_users")
	if err != nil {
		return err
	}

	// 使用匹配商户端的字段名
	payload := &entity.AdminmUserPayload{
		LoginName: req.Username, // control 用 username，商户端用 login_name
		Name:      req.Username, // 如果没有单独的 name，用 username 作为默认值
		Phone:     req.Phone,    // 手机号
		Password:  req.Password,
	}

	resp, err := doMerchantRequest("POST", url, payload)
	if err != nil {
		logx.Errorf("创建商户管理用户失败: merchant=%s, err=%v", req.MerchantNo, err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("创建失败，状态码: %d", resp.StatusCode)
	}

	logx.Infof("商户管理用户已创建: merchant=%s, username=%s", req.MerchantNo, req.Username)
	return nil
}

// UpdateAdminmUser 更新商户管理用户
func UpdateAdminmUser(req *model.UpdateAdminmUserReq) error {
	url, err := getMerchantAPIURL(req.MerchantNo, "/v1/control/adminm_users")
	if err != nil {
		return err
	}

	// 校验和规范化IP白名单
	var normalizedIPs *string
	if req.AllowedIPs != nil {
		normalized, err := normalizeAllowedIPs(*req.AllowedIPs)
		if err != nil {
			return fmt.Errorf("IP白名单格式错误: %v", err)
		}
		normalizedIPs = &normalized
	}

	// 使用匹配商户端的字段名
	payload := &entity.AdminmUserUpdatePayload{
		LoginName:  req.TargetUsername, // 用 login_name 定位用户
		Password:   req.Password,       // 更新密码
		AllowedIPs: normalizedIPs,      // 更新IP白名单（已规范化）
	}

	resp, err := doMerchantRequest("PUT", url, payload)
	if err != nil {
		logx.Errorf("更新商户管理用户失败: merchant=%s, err=%v", req.MerchantNo, err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("更新失败，状态码: %d", resp.StatusCode)
	}

	logx.Infof("商户管理用户已更新: merchant=%s, target=%s", req.MerchantNo, req.TargetUsername)
	return nil
}

// DeleteAdminmUser 删除商户管理用户
func DeleteAdminmUser(req *model.DeleteAdminmUserReq) error {
	url, err := getMerchantAPIURL(req.MerchantNo, "/v1/control/adminm_users")
	if err != nil {
		return err
	}

	// 使用匹配商户端的字段名
	payload := &entity.AdminmUserDeletePayload{
		LoginName: req.Username, // 用 login_name 定位用户
	}

	resp, err := doMerchantRequest("DELETE", url, payload)
	if err != nil {
		logx.Errorf("删除商户管理用户失败: merchant=%s, err=%v", req.MerchantNo, err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("删除失败，状态码: %d", resp.StatusCode)
	}

	logx.Infof("商户管理用户已删除: merchant=%s, username=%s", req.MerchantNo, req.Username)
	return nil
}

// QueryAdminmUsers 查询商户管理用户列表
func QueryAdminmUsers(merchantNo string, page, size int, username string) (*entity.AdminmUserQueryResp, error) {
	// 构建查询参数
	path := fmt.Sprintf("/v1/control/adminm_users?page=%d&size=%d", page, size)
	if username != "" {
		path += fmt.Sprintf("&username=%s", username)
	}

	url, err := getMerchantAPIURL(merchantNo, path)
	if err != nil {
		return &entity.AdminmUserQueryResp{Err: err.Error()}, nil
	}

	resp, err := doMerchantRequest("GET", url, nil)
	if err != nil {
		logx.Errorf("查询商户管理用户失败: merchant=%s, err=%v", merchantNo, err)
		return &entity.AdminmUserQueryResp{Err: fmt.Sprintf("请求失败: %v", err)}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return &entity.AdminmUserQueryResp{Err: fmt.Sprintf("请求失败，状态码: %d", resp.StatusCode)}, nil
	}

	var result entity.AdminmUserQueryResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return &entity.AdminmUserQueryResp{Err: fmt.Sprintf("解析响应失败: %v", err)}, nil
	}

	return &result, nil
}

// QueryAdminmActive 查询商户活跃数据
func QueryAdminmActive(merchantNo string) (*entity.AdminmActiveResp, error) {
	url, err := getMerchantAPIURL(merchantNo, "/v1/control/active")
	if err != nil {
		return &entity.AdminmActiveResp{Err: err.Error()}, nil
	}

	resp, err := doMerchantRequest("GET", url, nil)
	if err != nil {
		logx.Errorf("查询商户活跃数据失败: merchant=%s, err=%v", merchantNo, err)
		return &entity.AdminmActiveResp{Err: fmt.Sprintf("请求失败: %v", err)}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return &entity.AdminmActiveResp{Err: fmt.Sprintf("请求失败，状态码: %d", resp.StatusCode)}, nil
	}

	var result entity.AdminmActiveResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return &entity.AdminmActiveResp{Err: fmt.Sprintf("解析响应失败: %v", err)}, nil
	}

	return &result, nil
}

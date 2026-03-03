package deploy

import (
	"errors"
	"fmt"
	"server/internal/server/model"
	"server/pkg/dbs"
	"server/pkg/entity"
	"strings"
	"time"
)

// findRedisServerId 根据商户ID找到运行 tsdd-redis 的服务器ID
// 集群模式：从 cluster_nodes 表查找 db 角色节点
// 单机模式：如果没有集群节点，直接查找商户关联的服务器
func findRedisServerId(merchantId int) (int, error) {
	// 先查 cluster_nodes 表的 db 节点
	var dbNode entity.ClusterNodes
	has, err := dbs.DBAdmin.Where("merchant_id = ? AND node_role = ?", merchantId, entity.ClusterRoleDB).Get(&dbNode)
	if err != nil {
		return 0, fmt.Errorf("查询集群节点失败: %v", err)
	}
	if has {
		return dbNode.ServerId, nil
	}

	// 没有集群节点，说明是单机部署，查找商户关联的服务器
	var server entity.Servers
	has, err = dbs.DBAdmin.Where("merchant_id = ? AND status = 1", merchantId).Get(&server)
	if err != nil {
		return 0, fmt.Errorf("查询服务器失败: %v", err)
	}
	if !has {
		return 0, errors.New("未找到该商户的服务器")
	}
	return server.Id, nil
}

// redisCmd 在商户的 Redis 服务器上执行 redis-cli 命令（通过 docker exec tsdd-redis）
func redisCmd(serverId int, args string) (string, error) {
	client, err := GetSSHClient(serverId)
	if err != nil {
		return "", err
	}

	cmd := fmt.Sprintf("docker exec tsdd-redis redis-cli %s", args)
	output, err := client.ExecuteCommandWithTimeout(cmd, 10*time.Second)
	if err != nil {
		return "", fmt.Errorf("执行 Redis 命令失败: %v", err)
	}
	return strings.TrimSpace(output), nil
}

// GetRateLimitStatus 获取商户的限流状态
func GetRateLimitStatus(merchantId int) (*model.RateLimitStatusResp, error) {
	serverId, err := findRedisServerId(merchantId)
	if err != nil {
		return nil, err
	}

	// 获取限流开关状态
	enabledVal, err := redisCmd(serverId, "GET rl:global:enabled")
	if err != nil {
		return nil, fmt.Errorf("获取限流状态失败: %v", err)
	}

	// enabledVal == "0" 表示关闭限流，其他值（空/nil/"1"）表示开启
	enabled := enabledVal != "0"

	// 获取白名单
	whitelistStr, err := redisCmd(serverId, "SMEMBERS rl:whitelist")
	if err != nil {
		return nil, fmt.Errorf("获取白名单失败: %v", err)
	}

	var whitelist []string
	if whitelistStr != "" && whitelistStr != "(empty list or set)" && whitelistStr != "(empty array)" {
		for _, line := range strings.Split(whitelistStr, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			// 去掉序号前缀 "1) "
			if idx := strings.Index(line, ") "); idx >= 0 {
				line = line[idx+2:]
			}
			// 去掉引号
			line = strings.Trim(line, "\"")
			if line != "" {
				whitelist = append(whitelist, line)
			}
		}
	}

	if whitelist == nil {
		whitelist = []string{}
	}

	return &model.RateLimitStatusResp{
		Enabled:   enabled,
		Whitelist: whitelist,
	}, nil
}

// ToggleRateLimit 切换限流开关
func ToggleRateLimit(merchantId int, enabled bool) error {
	serverId, err := findRedisServerId(merchantId)
	if err != nil {
		return err
	}

	if enabled {
		_, err := redisCmd(serverId, "DEL rl:global:enabled")
		if err != nil {
			return fmt.Errorf("开启限流失败: %v", err)
		}
	} else {
		_, err := redisCmd(serverId, `SET rl:global:enabled 0`)
		if err != nil {
			return fmt.Errorf("关闭限流失败: %v", err)
		}
	}
	return nil
}

// AddWhitelistIP 添加白名单 IP
func AddWhitelistIP(merchantId int, ip string) error {
	serverId, err := findRedisServerId(merchantId)
	if err != nil {
		return err
	}

	_, err = redisCmd(serverId, fmt.Sprintf("SADD rl:whitelist %s", ip))
	if err != nil {
		return fmt.Errorf("添加白名单失败: %v", err)
	}
	return nil
}

// RemoveWhitelistIP 移除白名单 IP
func RemoveWhitelistIP(merchantId int, ip string) error {
	serverId, err := findRedisServerId(merchantId)
	if err != nil {
		return err
	}

	_, err = redisCmd(serverId, fmt.Sprintf("SREM rl:whitelist %s", ip))
	if err != nil {
		return fmt.Errorf("移除白名单失败: %v", err)
	}
	return nil
}

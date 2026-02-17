package deploy

import (
	"crypto/tls"
	"fmt"
	"net"
	"server/internal/server/model"
	"server/pkg/dbs"
	"server/pkg/entity"
	"server/pkg/gostapi"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/crypto/ssh"
)

const (
	maxTlsConcurrent = 5 // TLS 批量操作并发数
	certRemotePath   = "/etc/gost/certs"
)

// ========== 批量升级 TLS ==========

// BatchUpgradeTls 批量将系统服务器升级为 TLS 模式
func BatchUpgradeTls(req model.BatchUpgradeTlsReq) (*model.BatchTlsResp, error) {
	// 1. 获取证书
	var serverCert entity.TlsCertificates
	has, err := dbs.DBAdmin.Where("name = 'gost-server' AND status = 1").Get(&serverCert)
	if err != nil {
		return nil, fmt.Errorf("查询证书失败: %v", err)
	}
	if !has {
		return nil, fmt.Errorf("未找到有效的服务器证书，请先生成证书")
	}

	var caCert entity.TlsCertificates
	has, err = dbs.DBAdmin.Where("name = 'gost-ca' AND status = 1").Get(&caCert)
	if err != nil {
		return nil, fmt.Errorf("查询 CA 证书失败: %v", err)
	}
	if !has {
		return nil, fmt.Errorf("未找到有效的 CA 证书")
	}

	// 2. 获取目标服务器
	servers, err := getTargetSystemServers(req.ServerIds)
	if err != nil {
		return nil, err
	}
	if len(servers) == 0 {
		return nil, fmt.Errorf("未找到可升级的系统服务器")
	}

	// 3. 并发执行升级
	results := make([]model.TlsServerResult, len(servers))
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxTlsConcurrent)

	for i, server := range servers {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, s entity.Servers) {
			defer wg.Done()
			defer func() { <-sem }()
			results[idx] = upgradeServerTls(s, caCert, serverCert)
		}(i, server)
	}
	wg.Wait()

	// 4. 统计结果
	resp := &model.BatchTlsResp{
		Total:   len(results),
		Results: results,
	}
	for _, r := range results {
		if r.Success {
			resp.Success++
		} else {
			resp.Failed++
		}
	}

	logx.Infof("TLS 批量升级完成: 总计=%d, 成功=%d, 失败=%d", resp.Total, resp.Success, resp.Failed)
	return resp, nil
}

// upgradeServerTls 单台服务器 TLS 升级
func upgradeServerTls(server entity.Servers, caCert, serverCert entity.TlsCertificates) model.TlsServerResult {
	result := model.TlsServerResult{
		ServerId:   server.Id,
		ServerName: server.Name,
		Host:       server.Host,
	}

	// 1. SSH 推送证书
	err := pushCertsToServer(server, caCert, serverCert)
	if err != nil {
		result.Error = fmt.Sprintf("推送证书失败: %v", err)
		logx.Errorf("TLS upgrade: push certs to server %d(%s) err: %v", server.Id, server.Host, err)
		return result
	}

	// 2. 更新 GOST 服务 listener 为 TLS
	err = upgradeGostListenerToTls(server.Host)
	if err != nil {
		result.Error = fmt.Sprintf("更新 GOST 配置失败: %v", err)
		logx.Errorf("TLS upgrade: update gost config on server %d(%s) err: %v", server.Id, server.Host, err)
		return result
	}

	// 3. 更新数据库状态
	now := time.Now()
	_, err = dbs.DBAdmin.Where("id = ?", server.Id).Cols("tls_enabled", "tls_deployed_at", "updated_at").Update(&entity.Servers{
		TlsEnabled:    1,
		TlsDeployedAt: &now,
		UpdatedAt:     now,
	})
	if err != nil {
		result.Error = fmt.Sprintf("更新数据库状态失败: %v", err)
		logx.Errorf("TLS upgrade: update db for server %d err: %v", server.Id, err)
		return result
	}

	result.Success = true
	logx.Infof("TLS upgrade: server %d(%s) 升级成功", server.Id, server.Host)
	return result
}

// pushCertsToServer 通过 SSH 推送证书到服务器
func pushCertsToServer(server entity.Servers, caCert, serverCert entity.TlsCertificates) error {
	client, err := getSSHClientForServer(server)
	if err != nil {
		return fmt.Errorf("SSH 连接失败: %v", err)
	}
	defer client.Close()

	// 创建目录 + 写入证书文件
	script := fmt.Sprintf(`#!/bin/bash
set -e
if [ "$(id -u)" -eq 0 ]; then SUDO=""; else SUDO="sudo"; fi
$SUDO mkdir -p %s

$SUDO tee %s/ca.crt > /dev/null << 'CERTEOF'
%s
CERTEOF

$SUDO tee %s/server.crt > /dev/null << 'CERTEOF'
%s
CERTEOF

$SUDO tee %s/server.key > /dev/null << 'CERTEOF'
%s
CERTEOF

$SUDO chmod 600 %s/server.key
echo "证书部署完成"
`, certRemotePath,
		certRemotePath, caCert.CertPem,
		certRemotePath, serverCert.CertPem,
		certRemotePath, serverCert.KeyPem,
		certRemotePath)

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("创建 SSH 会话失败: %v", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(script)
	if err != nil {
		return fmt.Errorf("执行脚本失败: %v, output: %s", err, string(output))
	}

	return nil
}

// upgradeGostListenerToTls 通过 GOST API 将所有 tcp listener 改为 tls
func upgradeGostListenerToTls(serverIP string) error {
	config, err := gostapi.GetConfig(serverIP, "")
	if err != nil {
		return fmt.Errorf("获取 GOST 配置失败: %v", err)
	}

	for _, svc := range config.Services {
		if svc.Listener == nil || svc.Listener.Type != "tcp" {
			continue
		}
		// 只更新 tcp-relay 和 tcp-direct 类型的系统服务器转发服务
		// 跳过不相关的服务
		if svc.Handler == nil {
			continue
		}

		// 更新 listener 为 tls，并配置证书路径
		updatedSvc := &gostapi.ServiceConfig{
			Name: svc.Name,
			Addr: svc.Addr,
			Handler: svc.Handler,
			Listener: &gostapi.ListenerConfig{
				Type: "tls",
				TLS: &gostapi.TLSConfig{
					CertFile: certRemotePath + "/server.crt",
					KeyFile:  certRemotePath + "/server.key",
				},
			},
			Forwarder: svc.Forwarder,
		}

		_, err := gostapi.UpdateService(serverIP, svc.Name, updatedSvc)
		if err != nil {
			return fmt.Errorf("更新服务 %s 失败: %v", svc.Name, err)
		}
	}

	// 持久化配置
	_, err = gostapi.SaveConfig(serverIP, "yaml", "")
	if err != nil {
		return fmt.Errorf("保存配置失败: %v", err)
	}

	return nil
}

// ========== 批量回滚 TLS ==========

// BatchRollbackTls 批量将系统服务器回滚为 TCP 模式
func BatchRollbackTls(req model.BatchRollbackTlsReq) (*model.BatchTlsResp, error) {
	servers, err := getTargetSystemServers(req.ServerIds)
	if err != nil {
		return nil, err
	}
	if len(servers) == 0 {
		return nil, fmt.Errorf("未找到可回滚的系统服务器")
	}

	results := make([]model.TlsServerResult, len(servers))
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxTlsConcurrent)

	for i, server := range servers {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, s entity.Servers) {
			defer wg.Done()
			defer func() { <-sem }()
			results[idx] = rollbackServerTls(s)
		}(i, server)
	}
	wg.Wait()

	resp := &model.BatchTlsResp{
		Total:   len(results),
		Results: results,
	}
	for _, r := range results {
		if r.Success {
			resp.Success++
		} else {
			resp.Failed++
		}
	}

	logx.Infof("TLS 批量回滚完成: 总计=%d, 成功=%d, 失败=%d", resp.Total, resp.Success, resp.Failed)
	return resp, nil
}

// rollbackServerTls 单台服务器 TLS 回滚
func rollbackServerTls(server entity.Servers) model.TlsServerResult {
	result := model.TlsServerResult{
		ServerId:   server.Id,
		ServerName: server.Name,
		Host:       server.Host,
	}

	// 1. 通过 GOST API 将所有 tls listener 改回 tcp
	err := rollbackGostListenerToTcp(server.Host)
	if err != nil {
		result.Error = fmt.Sprintf("回滚 GOST 配置失败: %v", err)
		logx.Errorf("TLS rollback: server %d(%s) err: %v", server.Id, server.Host, err)
		return result
	}

	// 2. 更新数据库状态
	_, err = dbs.DBAdmin.Where("id = ?", server.Id).Cols("tls_enabled", "updated_at").Update(&entity.Servers{
		TlsEnabled: 0,
		UpdatedAt:  time.Now(),
	})
	if err != nil {
		result.Error = fmt.Sprintf("更新数据库状态失败: %v", err)
		logx.Errorf("TLS rollback: update db for server %d err: %v", server.Id, err)
		return result
	}

	result.Success = true
	logx.Infof("TLS rollback: server %d(%s) 回滚成功", server.Id, server.Host)
	return result
}

// rollbackGostListenerToTcp 将所有 tls listener 改回 tcp
func rollbackGostListenerToTcp(serverIP string) error {
	config, err := gostapi.GetConfig(serverIP, "")
	if err != nil {
		return fmt.Errorf("获取 GOST 配置失败: %v", err)
	}

	for _, svc := range config.Services {
		if svc.Listener == nil || svc.Listener.Type != "tls" {
			continue
		}
		// 跳过商户本地转发服务 (local-*)，这些 tls listener 是 relay+tls 的一部分
		if len(svc.Name) > 6 && svc.Name[:6] == "local-" {
			continue
		}

		updatedSvc := &gostapi.ServiceConfig{
			Name: svc.Name,
			Addr: svc.Addr,
			Handler: svc.Handler,
			Listener: &gostapi.ListenerConfig{
				Type: "tcp",
			},
			Forwarder: svc.Forwarder,
		}

		_, err := gostapi.UpdateService(serverIP, svc.Name, updatedSvc)
		if err != nil {
			return fmt.Errorf("回滚服务 %s 失败: %v", svc.Name, err)
		}
	}

	_, err = gostapi.SaveConfig(serverIP, "yaml", "")
	if err != nil {
		return fmt.Errorf("保存配置失败: %v", err)
	}

	return nil
}

// ========== TLS 状态查询 ==========

// GetTlsStatus 获取所有系统服务器的 TLS 状态
func GetTlsStatus() (*model.TlsStatusResp, error) {
	var servers []entity.Servers
	err := dbs.DBAdmin.Where("server_type = 2 AND status = 1").Find(&servers)
	if err != nil {
		return nil, fmt.Errorf("查询系统服务器失败: %v", err)
	}

	resp := &model.TlsStatusResp{
		Total:   len(servers),
		Servers: make([]model.TlsServerStatus, 0, len(servers)),
	}

	for _, s := range servers {
		status := model.TlsServerStatus{
			ServerId:   s.Id,
			ServerName: s.Name,
			Host:       s.Host,
			TlsEnabled: s.TlsEnabled,
		}
		if s.TlsDeployedAt != nil {
			status.TlsDeployedAt = s.TlsDeployedAt.Format("2006-01-02 15:04:05")
		}

		if s.TlsEnabled == 1 {
			resp.TlsCount++
		} else {
			resp.TcpCount++
		}

		resp.Servers = append(resp.Servers, status)
	}

	return resp, nil
}

// VerifyTlsStatus 验证系统服务器的 TLS 连接是否正常
func VerifyTlsStatus() (*model.TlsStatusResp, error) {
	var servers []entity.Servers
	err := dbs.DBAdmin.Where("server_type = 2 AND status = 1").Find(&servers)
	if err != nil {
		return nil, fmt.Errorf("查询系统服务器失败: %v", err)
	}

	resp := &model.TlsStatusResp{
		Total:   len(servers),
		Servers: make([]model.TlsServerStatus, len(servers)),
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, maxTlsConcurrent)

	for i, s := range servers {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, server entity.Servers) {
			defer wg.Done()
			defer func() { <-sem }()

			status := model.TlsServerStatus{
				ServerId:   server.Id,
				ServerName: server.Name,
				Host:       server.Host,
				TlsEnabled: server.TlsEnabled,
			}
			if server.TlsDeployedAt != nil {
				status.TlsDeployedAt = server.TlsDeployedAt.Format("2006-01-02 15:04:05")
			}

			if server.TlsEnabled == 1 {
				resp.TlsCount++
				// 验证 TLS 连接 — 尝试连到 GOST 的第一个端口做 TLS 握手
				err := verifyTlsConnection(server.Host)
				if err != nil {
					status.TlsVerified = false
					status.VerifyError = err.Error()
				} else {
					status.TlsVerified = true
				}
			} else {
				resp.TcpCount++
				status.TlsVerified = false
				status.VerifyError = "TLS 未启用"
			}

			resp.Servers[idx] = status
		}(i, s)
	}
	wg.Wait()

	return resp, nil
}

// verifyTlsConnection 验证 TLS 连接
func verifyTlsConnection(host string) error {
	// 获取该服务器上的第一个 GOST 服务端口
	config, err := gostapi.GetConfig(host, "")
	if err != nil {
		return fmt.Errorf("获取 GOST 配置失败: %v", err)
	}

	// 找一个 tls listener 的端口测试
	for _, svc := range config.Services {
		if svc.Listener != nil && svc.Listener.Type == "tls" && svc.Addr != "" {
			addr := svc.Addr
			if addr[0] == ':' {
				addr = host + addr
			}
			conn, err := tls.DialWithDialer(
				&net.Dialer{Timeout: 5 * time.Second},
				"tcp",
				addr,
				&tls.Config{InsecureSkipVerify: true},
			)
			if err != nil {
				return fmt.Errorf("TLS 握手失败(%s): %v", addr, err)
			}
			conn.Close()
			return nil
		}
	}

	return fmt.Errorf("未找到 TLS 监听服务")
}

// ========== 辅助函数 ==========

// getTargetSystemServers 获取目标系统服务器列表
func getTargetSystemServers(serverIds []int) ([]entity.Servers, error) {
	var servers []entity.Servers
	session := dbs.DBAdmin.Where("server_type = 2 AND status = 1")

	if len(serverIds) > 0 {
		session = session.In("id", serverIds)
	}

	err := session.Find(&servers)
	if err != nil {
		return nil, fmt.Errorf("查询系统服务器失败: %v", err)
	}
	return servers, nil
}

// getSSHClientForServer 获取服务器的 SSH 客户端
func getSSHClientForServer(server entity.Servers) (*ssh.Client, error) {
	var authMethods []ssh.AuthMethod
	if server.Password != "" {
		authMethods = append(authMethods, ssh.Password(server.Password))
	}
	if server.PrivateKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(server.PrivateKey))
		if err != nil {
			return nil, fmt.Errorf("解析私钥失败: %v", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}
	if len(authMethods) == 0 {
		return nil, fmt.Errorf("无可用的认证方式")
	}

	config := &ssh.ClientConfig{
		User:            server.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	port := server.Port
	if port == 0 {
		port = 22
	}

	return ssh.Dial("tcp", fmt.Sprintf("%s:%d", server.Host, port), config)
}
package deploy

import (
	"fmt"
	"net"
	"server/pkg/dbs"
	"server/pkg/entity"
	"server/pkg/gostapi"
	"sync"
	"time"
)

// ConnectionCheckResult 连接检测结果
type ConnectionCheckResult struct {
	Merchants []MerchantConnectionStatus `json:"merchants"`
}

// MerchantConnectionStatus 商户连接状态
type MerchantConnectionStatus struct {
	MerchantId   int                      `json:"merchant_id"`
	MerchantName string                   `json:"merchant_name"`
	ServerIP     string                   `json:"server_ip"`
	GostServers  []GostServerStatus       `json:"gost_servers"`
	Services     []MerchantServiceStatus  `json:"services"`
}

// GostServerStatus GOST 系统服务器状态
type GostServerStatus struct {
	RelationId   int    `json:"relation_id"`
	ServerId     int    `json:"server_id"`
	ServerName   string `json:"server_name"`
	Host         string `json:"host"`
	GostAPI      string `json:"gost_api"`      // ok / offline / error
	ServiceCount int    `json:"service_count"`
	ChainCount   int    `json:"chain_count"`
	Port443      string `json:"port_443"`      // ok / fail
	Port80       string `json:"port_80"`       // ok / fail
	Port8080     string `json:"port_8080"`     // ok / fail
}

// MerchantServiceStatus 商户服务端口状态
type MerchantServiceStatus struct {
	Port   int    `json:"port"`
	Name   string `json:"name"`
	Status string `json:"status"` // ok / fail
}

// ConnectionCheck 检测所有商户的 GOST 和服务器连接状态（并发）
func ConnectionCheck() (*ConnectionCheckResult, error) {
	var merchants []entity.Merchants
	if err := dbs.DBAdmin.Where("status = 1").Find(&merchants); err != nil {
		return nil, fmt.Errorf("查询商户失败: %w", err)
	}

	result := &ConnectionCheckResult{
		Merchants: make([]MerchantConnectionStatus, len(merchants)),
	}

	var wg sync.WaitGroup
	for i, m := range merchants {
		wg.Add(1)
		go func(idx int, merchant entity.Merchants) {
			defer wg.Done()
			result.Merchants[idx] = checkOneMerchant(merchant)
		}(i, m)
	}
	wg.Wait()

	return result, nil
}

// ConnectionCheckByMerchant 检测单个商户的连接状态
func ConnectionCheckByMerchant(merchantId int) (*MerchantConnectionStatus, error) {
	var merchant entity.Merchants
	has, err := dbs.DBAdmin.Where("id = ? AND status = 1", merchantId).Get(&merchant)
	if err != nil {
		return nil, fmt.Errorf("查询商户失败: %w", err)
	}
	if !has {
		return nil, fmt.Errorf("商户不存在或已禁用")
	}

	ms := checkOneMerchant(merchant)
	return &ms, nil
}

// checkOneMerchant 检测单个商户的所有连接
func checkOneMerchant(m entity.Merchants) MerchantConnectionStatus {
	ms := MerchantConnectionStatus{
		MerchantId:   m.Id,
		MerchantName: m.Name,
		ServerIP:     m.ServerIP,
	}

	// 检测商户服务端口
	// 内部端口（5100/8090）不应公网可达，fail 标记为 protected
	ms.Services = []MerchantServiceStatus{
		{Port: 10443, Name: "GOST-IM", Status: tcpCheck(m.ServerIP, 10443)},
		{Port: 10080, Name: "GOST-HTTP", Status: tcpCheck(m.ServerIP, 10080)},
		{Port: 10800, Name: "GOST-File", Status: tcpCheck(m.ServerIP, 10800)},
		{Port: 5100, Name: "WuKongIM", Status: internalPortCheck(m.ServerIP, 5100)},
		{Port: 8090, Name: "tsdd-server", Status: internalPortCheck(m.ServerIP, 8090)},
	}

	// 检测关联的 GOST 系统服务器
	var relations []entity.MerchantGostServers
	dbs.DBAdmin.Where("merchant_id = ? AND status = 1", m.Id).Find(&relations)

	ms.GostServers = make([]GostServerStatus, 0, len(relations))
	for _, rel := range relations {
		var server entity.Servers
		has, _ := dbs.DBAdmin.Where("id = ?", rel.ServerId).Get(&server)
		if !has {
			continue
		}

		gs := GostServerStatus{
			RelationId: rel.Id,
			ServerId:   server.Id,
			ServerName: server.Name,
			Host:       server.Host,
			Port443:    tcpCheck(server.Host, 443),
			Port80:     tcpCheck(server.Host, 80),
			Port8080:   tcpCheck(server.Host, 8080),
		}

		// 检测 GOST API（用 GetServiceList/GetChainList 获取完整列表，含动态创建的）
		svcList, svcErr := gostapi.GetServiceList(server.Host)
		chainList, chainErr := gostapi.GetChainList(server.Host)
		if svcErr != nil && chainErr != nil {
			gs.GostAPI = "offline"
		} else {
			gs.GostAPI = "ok"
			if svcErr == nil {
				gs.ServiceCount = svcList.Count
			}
			if chainErr == nil {
				gs.ChainCount = chainList.Count
			}
		}

		ms.GostServers = append(ms.GostServers, gs)
	}

	return ms
}

func internalPortCheck(host string, port int) string {
	status := tcpCheck(host, port)
	if status == "fail" {
		return "protected"
	}
	return status
}

// ConnectionCheckByServer 检测单个系统服务器的 GOST 状态
func ConnectionCheckByServer(serverId int) (*GostServerStatus, error) {
	var server entity.Servers
	has, err := dbs.DBAdmin.Where("id = ? AND status = 1", serverId).Get(&server)
	if err != nil {
		return nil, fmt.Errorf("查询服务器失败: %w", err)
	}
	if !has {
		return nil, fmt.Errorf("服务器不存在或已停用")
	}

	gs := GostServerStatus{
		ServerId:   server.Id,
		ServerName: server.Name,
		Host:       server.Host,
		Port443:    tcpCheck(server.Host, 443),
		Port80:     tcpCheck(server.Host, 80),
		Port8080:   tcpCheck(server.Host, 8080),
	}

	svcList, svcErr := gostapi.GetServiceList(server.Host)
	chainList, chainErr := gostapi.GetChainList(server.Host)
	if svcErr != nil && chainErr != nil {
		gs.GostAPI = "offline"
	} else {
		gs.GostAPI = "ok"
		if svcErr == nil {
			gs.ServiceCount = svcList.Count
		}
		if chainErr == nil {
			gs.ChainCount = chainList.Count
		}
	}

	return &gs, nil
}

func tcpCheck(host string, port int) string {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 3*time.Second)
	if err != nil {
		return "fail"
	}
	conn.Close()
	return "ok"
}

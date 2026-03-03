package merchant

import (
	"fmt"
	"net"
	"server/internal/server/model"
	"server/pkg/dbs"
	"server/pkg/entity"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

const maxTurnBatchConcurrent = 10

// TurnConfig TURN 完整配置
type TurnConfig struct {
	Server     string
	Username   string
	Credential string
}

// ListMerchantTurnConfigs 获取所有商户的 TURN 配置列表
func ListMerchantTurnConfigs(nameFilter string) ([]model.MerchantTurnConfigItem, error) {
	var merchants []entity.Merchants
	session := dbs.DBAdmin.NewSession()
	defer session.Close()

	if nameFilter != "" {
		session = session.Where("name LIKE ?", "%"+nameFilter+"%")
	}
	if err := session.OrderBy("id ASC").Find(&merchants); err != nil {
		return nil, fmt.Errorf("查询商户列表失败: %v", err)
	}

	items := make([]model.MerchantTurnConfigItem, len(merchants))
	for i, m := range merchants {
		var turnServer, turnUsername, turnCredential string
		if m.PackageConfiguration != nil {
			turnServer = m.PackageConfiguration.TurnServer
			turnUsername = m.PackageConfiguration.TurnUsername
			turnCredential = m.PackageConfiguration.TurnCredential
		}
		items[i] = model.MerchantTurnConfigItem{
			MerchantId:     m.Id,
			MerchantNo:     m.No,
			MerchantName:   m.Name,
			ServerIP:       m.ServerIP,
			Status:         m.Status,
			TurnServer:     turnServer,
			TurnUsername:   turnUsername,
			TurnCredential: turnCredential,
			UpdatedAt:      m.UpdatedAt.Format(time.DateTime),
		}
	}
	return items, nil
}

// UpdateMerchantTurnServer 更新单个商户的 TURN 配置并推送
func UpdateMerchantTurnServer(merchantId int, cfg TurnConfig) (*model.BatchTurnUpdateResult, error) {
	if err := validateTurnServer(cfg.Server); err != nil {
		return nil, err
	}

	var m entity.Merchants
	has, err := dbs.DBAdmin.ID(merchantId).Get(&m)
	if err != nil {
		return nil, fmt.Errorf("查询商户失败: %v", err)
	}
	if !has {
		return nil, fmt.Errorf("商户不存在: %d", merchantId)
	}

	result := updateSingleMerchantTurn(&m, cfg)
	if !result.Success {
		return &result, fmt.Errorf(result.Message)
	}
	return &result, nil
}

// BatchUpdateTurnServer 批量更新商户 TURN 配置
func BatchUpdateTurnServer(req model.BatchUpdateTurnServerReq) (*model.BatchUpdateTurnServerResp, error) {
	if err := validateTurnServer(req.TurnServer); err != nil {
		return nil, err
	}

	cfg := TurnConfig{
		Server:     req.TurnServer,
		Username:   req.TurnUsername,
		Credential: req.TurnCredential,
	}

	var merchants []entity.Merchants
	if err := dbs.DBAdmin.In("id", req.MerchantIds).Find(&merchants); err != nil {
		return nil, fmt.Errorf("查询商户列表失败: %v", err)
	}
	if len(merchants) == 0 {
		return nil, fmt.Errorf("未找到有效的商户")
	}

	resp := &model.BatchUpdateTurnServerResp{
		TotalCount: len(merchants),
		Results:    make([]model.BatchTurnUpdateResult, 0, len(merchants)),
	}

	sem := make(chan struct{}, maxTurnBatchConcurrent)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, m := range merchants {
		merchant := m
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			r := updateSingleMerchantTurn(&merchant, cfg)

			mu.Lock()
			resp.Results = append(resp.Results, r)
			if r.Success {
				resp.SuccessCount++
			} else {
				resp.FailCount++
			}
			mu.Unlock()
		}()
	}

	wg.Wait()
	return resp, nil
}

// updateSingleMerchantTurn 更新单个商户的 TURN 并推送配置
func updateSingleMerchantTurn(merchant *entity.Merchants, cfg TurnConfig) model.BatchTurnUpdateResult {
	result := model.BatchTurnUpdateResult{
		MerchantId:   merchant.Id,
		MerchantName: merchant.Name,
		ServerIP:     merchant.ServerIP,
	}

	if merchant.PackageConfiguration == nil {
		merchant.PackageConfiguration = &entity.PackageConfiguration{}
	}

	oldTurn := merchant.PackageConfiguration.TurnServer
	merchant.PackageConfiguration.TurnServer = cfg.Server
	merchant.PackageConfiguration.TurnUsername = cfg.Username
	merchant.PackageConfiguration.TurnCredential = cfg.Credential
	merchant.UpdatedAt = time.Now()

	_, err := dbs.DBAdmin.ID(merchant.Id).
		Cols("package_configuration", "updated_at").
		Update(merchant)
	if err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("数据库更新失败: %v", err)
		return result
	}

	if merchant.ServerIP == "" {
		result.Success = true
		result.Message = fmt.Sprintf("DB已更新(%s->%s)，商户无服务器IP，跳过推送", oldTurn, cfg.Server)
		return result
	}

	if err := NotifyConfigUpdate(merchant); err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("配置推送失败(DB已更新 %s->%s): %v", oldTurn, cfg.Server, err)
		logx.Errorf("TURN配置推送失败: merchant=%d, err=%v", merchant.Id, err)
		return result
	}

	result.Success = true
	result.Message = fmt.Sprintf("更新成功: %s -> %s", oldTurn, cfg.Server)
	return result
}

// validateTurnServer 验证 TURN 服务器地址格式 (ip:port)
func validateTurnServer(turnServer string) error {
	if turnServer == "" {
		return fmt.Errorf("turn_server 不能为空")
	}
	host, port, err := net.SplitHostPort(turnServer)
	if err != nil {
		return fmt.Errorf("turn_server 格式错误，应为 ip:port: %v", err)
	}
	if net.ParseIP(host) == nil {
		return fmt.Errorf("turn_server IP 地址无效: %s", host)
	}
	if port == "" || port == "0" {
		return fmt.Errorf("turn_server 端口无效: %s", port)
	}
	return nil
}

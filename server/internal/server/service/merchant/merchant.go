package merchant

import (
	"fmt"
	"server/internal/dbhelper"
	"server/internal/server/model"
	utilSvc "server/internal/server/service/utils"
	"server/pkg/consts"
	"server/pkg/dbs"
	"server/pkg/entity"
	"server/pkg/gostapi"
	"strconv"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"xorm.io/xorm"
)

// ListMerchant 获取商户列表
func ListMerchant(page, size int, name, orderBy string, expiringSoon int, merchantNo string) ([]*model.Merchant, int64, error) {
	// 1. 获取商户列表
	merchants, total, err := dbhelper.FindMerchantListWithCondition(page, size, name, orderBy, expiringSoon, merchantNo)
	if err != nil {
		return nil, 0, err
	}
	// 如果没有商户，直接返回空列表
	if len(merchants) == 0 {
		return []*model.Merchant{}, total, nil
	}

	// 2. 获取所有商户ID
	merchantIds := make([]int, len(merchants))
	for i, m := range merchants {
		merchantIds[i] = m.Id
	}

	// 3. 批量查询 OSS 配置数量
	ossCountMap := getMerchantOssConfigCounts(merchantIds)

	// 4. 批量查询 GOST 服务器数量
	gostCountMap := getMerchantGostServerCounts(merchantIds)

	// 5. 构建返回结果
	merchantList := make([]*model.Merchant, len(merchants))
	for i, m := range merchants {
		val := &model.Merchant{}
		val.Init(&m)
		val.OssConfigCount = ossCountMap[m.Id]
		val.GostServerCount = gostCountMap[m.Id]
		merchantList[i] = val
	}

	return merchantList, total, nil
}

// getMerchantOssConfigCounts 批量获取商户 OSS 配置数量
func getMerchantOssConfigCounts(merchantIds []int) map[int]int {
	result := make(map[int]int)
	if len(merchantIds) == 0 {
		return result
	}

	type countResult struct {
		MerchantId int `xorm:"merchant_id"`
		Count      int `xorm:"cnt"`
	}
	var counts []countResult

	err := dbs.DBAdmin.Table("merchant_oss_configs").
		Select("merchant_id, COUNT(*) as cnt").
		In("merchant_id", merchantIds).
		Where("status = 1").
		GroupBy("merchant_id").
		Find(&counts)

	if err != nil {
		logx.Errorf("get merchant oss config counts error: %v", err)
		return result
	}

	for _, c := range counts {
		result[c.MerchantId] = c.Count
	}
	return result
}

// getMerchantGostServerCounts 批量获取商户 GOST 服务器数量
func getMerchantGostServerCounts(merchantIds []int) map[int]int {
	result := make(map[int]int)
	if len(merchantIds) == 0 {
		return result
	}

	type countResult struct {
		MerchantId int `xorm:"merchant_id"`
		Count      int `xorm:"cnt"`
	}
	var counts []countResult

	err := dbs.DBAdmin.Table("merchant_gost_servers").
		Select("merchant_id, COUNT(*) as cnt").
		In("merchant_id", merchantIds).
		Where("status = 1").
		GroupBy("merchant_id").
		Find(&counts)

	if err != nil {
		logx.Errorf("get merchant gost server counts error: %v", err)
		return result
	}

	for _, c := range counts {
		result[c.MerchantId] = c.Count
	}
	return result
}

// CreateMerchant 创建商户，返回商户ID
func CreateMerchant(req *model.CreateOrEditMerchantReq) (int, error) {
	// 1) 校验端口 ,10000是属于测试服的
	if req.Port < 10000 {
		return 0, fmt.Errorf("port 不能小于10000")
	}
	if req.Port > 65535 {
		return 0, fmt.Errorf("port 不能大于65535")
	}
	// 1.1) 校验 server_ip 必填
	serverIP := strings.TrimSpace(req.ServerIP)
	if serverIP == "" {
		return 0, fmt.Errorf("server_ip 为必填")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return 0, fmt.Errorf("name 为必填")
	}
	// 2) 实时计算 No
	req.No = utilSvc.Port2Enterprise(uint16(req.Port))
	// 3) 端口占用检查已移除：每个商户配独立系统服务器+隧道，端口可复用
	if req.Status == 0 {
		req.Status = 1 // 默认状态为正常
	}
	now := time.Now()
	if req.ExpiredAt == "" {
		// 默认过期时间为30天后
		req.ExpiredAt = now.AddDate(0, 0, 30).Format(time.DateTime)
	}

	// 校验：创建商户要求提供AWS云账号（二选一：手填或选择系统账号）
	var selectedAccount *entity.CloudAccounts
	if req.SelectedAwsAccountId > 0 {
		// 使用选中的系统账号
		var acc entity.CloudAccounts
		has, err := dbs.DBAdmin.Where("id = ? AND account_type = ? AND cloud_type = ? AND status = 1",
			req.SelectedAwsAccountId, "system", "aws").Get(&acc)
		if err != nil {
			return 0, fmt.Errorf("查询系统AWS账号失败: %v", err)
		}
		if !has {
			return 0, fmt.Errorf("选中的系统AWS账号不存在或不可用")
		}
		selectedAccount = &acc
	} else {
		// 手动填写账号
		if strings.TrimSpace(req.AwsAccessKeyId) == "" || strings.TrimSpace(req.AwsAccessKeySecret) == "" {
			return 0, fmt.Errorf("aws_access_key_id 与 aws_access_key_secret 为必填，或选择系统AWS账号")
		}
	}

	expiredAt, err := time.ParseInLocation(time.DateTime, req.ExpiredAt, time.Local)
	if err != nil {
		return 0, err
	}
	if expiredAt.Before(now) {
		return 0, fmt.Errorf("expired_at 不能小于当前时间")
	}
	if req.PackageConfiguration != nil {
		req.PackageConfiguration.ExpiredAt = expiredAt.Unix()
	}
	merchant := &entity.Merchants{
		No:                   req.No,
		ServerIP:             serverIP,
		Port:                 req.Port,
		Name:                 name,
		AppName:              strings.TrimSpace(req.AppName),
		LogoUrl:              strings.TrimSpace(req.LogoUrl),
		IconUrl:              strings.TrimSpace(req.IconUrl),
		Status:               req.Status,
		PackageConfiguration: req.PackageConfiguration,
		ExpiredAt:            expiredAt,
	}

	// 事务：创建商户并写入 CloudAccounts（类型为 merchant）
	if err := dbs.DBAdmin.WithTx(func(session *xorm.Session) (err error) {
		merchant.CreatedAt = now
		merchant.UpdatedAt = now

		if _, err = session.Insert(merchant); err != nil {
			return err
		}

		// 商户创建成功后：自动创建商户服务器记录
		merchantServer := &entity.Servers{
			ServerType:  1, // 商户服务器
			MerchantId:  merchant.Id,
			Name:        fmt.Sprintf("%s-商户服务器", merchant.Name),
			Host:        req.ServerIP,
			Port:        consts.DefaultPort,
			Username:    consts.DefaultUsername,
			AuthType:    1, // 密码认证
			Password:    consts.DefaultPassword,
			PrivateKey:  "",
			DeployPath:  consts.DeployPath,
			Status:      1,
			Description: "商户服务器-自动创建",
			Tags:        "",
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if _, err = session.Insert(merchantServer); err != nil {
			return err
		}

		// 处理 AWS 云账号：三种情况
		if selectedAccount != nil {
			// 选择了系统账号
			if req.RemoveFromSystem {
				// 将系统账号转换为商户账号
				_, err = session.Exec(
					"UPDATE cloud_accounts SET account_type = ?, merchant_id = ?, name = ?, description = ?, updated_at = now() WHERE id = ?",
					"merchant", merchant.Id, fmt.Sprintf("%s-aws", merchant.Name),
					fmt.Sprintf("商户:%s (从系统账号转换)", merchant.Name), selectedAccount.Id,
				)
				if err != nil {
					return fmt.Errorf("转换系统账号失败: %v", err)
				}
			} else {
				// 复制系统账号创建新的商户账号
				account := entity.CloudAccounts{
					AccountType:     "merchant",
					MerchantId:      merchant.Id,
					Name:            fmt.Sprintf("%s-aws", merchant.Name),
					CloudType:       "aws",
					AccessKeyId:     selectedAccount.AccessKeyId,
					AccessKeySecret: selectedAccount.AccessKeySecret,
					Description:     fmt.Sprintf("商户:%s (从系统账号复制)", merchant.Name),
					Status:          1,
					CreatedAt:       now,
					UpdatedAt:       now,
				}
				if _, err = session.Insert(&account); err != nil {
					return err
				}
			}
		} else {
			// 手动填写的账号
			account := entity.CloudAccounts{
				AccountType:     "merchant",
				MerchantId:      merchant.Id,
				Name:            fmt.Sprintf("%s-aws", merchant.Name),
				CloudType:       "aws",
				AccessKeyId:     req.AwsAccessKeyId,
				AccessKeySecret: req.AwsAccessKeySecret,
				Description:     fmt.Sprintf("商户:%s", merchant.Name),
				Status:          1,
				CreatedAt:       now,
				UpdatedAt:       now,
			}
			if _, err = session.Insert(&account); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return 0, err
	}

	// 为商户服务器创建本地 Gost 转发服务（通过任务队列，支持重试）
	if err := gostapi.EnqueueCreateMerchantLocalForwards(req.ServerIP); err != nil {
		logx.Errorf("enqueue create merchant local forwards task failed: %+v", err)
	}

	// 为系统服务器创建 Gost 服务（通过任务队列，支持重试）
	enqueueGostServicesForServers(req.Port, req.ServerIP, merchant.TunnelIP)

	return merchant.Id, nil
}

// enqueueGostServicesForServers 为所有系统服务器入队创建商户转发任务
// 根据每个系统服务器的 forward_type 选择加密或直连转发
// tunnelIP: 商户在系统服务器上分配的隧道 IP（多商户隔离）
func enqueueGostServicesForServers(listenPort int, forwardHost string, tunnelIP string) {
	var servers []entity.Servers
	if err := dbs.DBAdmin.Where("server_type = ? AND status = 1", 2).Find(&servers); err != nil {
		logx.Errorf("query selected servers err: %+v", err)
		return
	}

	if len(servers) == 0 {
		logx.Infof("no valid system servers found for gost service creation")
		return
	}

	encryptedCount := 0
	directCount := 0
	for _, s := range servers {
		var err error
		tlsEnabled := s.TlsEnabled == 1
		if s.ForwardType == entity.ForwardTypeDirect {
			// 直连转发：直接转发到商户业务程序 10000/10001/10002
			if tlsEnabled {
				err = gostapi.EnqueueCreateMerchantDirectForwardsWithTls(s.Host, listenPort, forwardHost, tunnelIP)
			} else {
				err = gostapi.EnqueueCreateMerchantDirectForwards(s.Host, listenPort, forwardHost, tunnelIP)
			}
			directCount++
		} else {
			// 加密转发（默认）：通过 relay+tls 转发到商户 GOST 10010/10011/10012
			if tlsEnabled {
				err = gostapi.EnqueueCreateMerchantForwardsWithTls(s.Host, listenPort, forwardHost, tunnelIP)
			} else {
				err = gostapi.EnqueueCreateMerchantForwards(s.Host, listenPort, forwardHost, tunnelIP)
			}
			encryptedCount++
		}
		if err != nil {
			logx.Errorf("enqueue create merchant forwards task for server %d (%s, forward_type=%d) failed: %+v",
				s.Id, s.Host, s.ForwardType, err)
		}
	}
	logx.Infof("enqueued create merchant forwards tasks for %d servers (encrypted: %d, direct: %d), port %d",
		len(servers), encryptedCount, directCount, listenPort)
}

// UpdateMerchant 更新商户
func UpdateMerchant(idStr string, req *model.CreateOrEditMerchantReq) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}

	merchant, err := dbhelper.GetMerchantByID(int(id))
	if err != nil {
		return err
	}

	merchant.Id = id
	oldServerIP := merchant.ServerIP
	ipChanged := false
	if req.Name != "" {
		merchant.Name = req.Name
	}
	// 更新应用名称（可选）
	if req.AppName != "" {
		merchant.AppName = strings.TrimSpace(req.AppName)
	}
	// 更新 Logo 和图标地址（可选）
	if req.LogoUrl != "" {
		merchant.LogoUrl = strings.TrimSpace(req.LogoUrl)
	}
	if req.IconUrl != "" {
		merchant.IconUrl = strings.TrimSpace(req.IconUrl)
	}
	if req.Status != 0 {
		merchant.Status = req.Status
	}
	// 允许更新 server_ip（如果提供）
	serverIp := strings.TrimSpace(req.ServerIP)
	if serverIp != "" {
		if serverIp != oldServerIP {
			ipChanged = true
		}
		merchant.ServerIP = serverIp
	}
	// 禁止修改 No；允许修改 Port，但需要校验占用
	// Merchants.CloudAccount 已弃用，忽略请求中的该字段
	if req.PackageConfiguration != nil {
		expiredAt, err := time.ParseInLocation(time.DateTime, req.ExpiredAt, time.Local)
		if err != nil {
			return err
		}
		req.PackageConfiguration.ExpiredAt = expiredAt.Unix()
		merchant.PackageConfiguration = req.PackageConfiguration
	}
	if req.ExpiredAt != "" {
		loc, _ := time.LoadLocation("Asia/Shanghai")
		expiredAt, err := time.ParseInLocation(time.DateTime, req.ExpiredAt, loc)
		if err != nil {
			return err
		}
		merchant.ExpiredAt = expiredAt
	}

	// 忽略端口修改（编辑商户时不允许修改端口）
	// 即使请求中带有 port，也不变更 merchant.Port 与 merchant.No
	// 在一个事务中更新商户与其 AWS 云账号
	if err := dbs.DBAdmin.WithTx(func(session *xorm.Session) error {
		// 更新 merchants
		if _, err := session.ID(merchant.Id).Update(merchant); err != nil {
			return err
		}

		// 若请求携带了新的 AWS 凭证，则对 cloud_accounts 做 upsert（account_type, merchant_id, cloud_type 唯一）
		accessKeyId := strings.TrimSpace(req.AwsAccessKeyId)
		accessKeySecret := strings.TrimSpace(req.AwsAccessKeySecret)
		if accessKeyId != "" && accessKeySecret != "" {
			updateSQL := "UPDATE cloud_accounts SET access_key_id = ?, access_key_secret = ?, updated_at = NOW() WHERE account_type = ? AND merchant_id = ? AND cloud_type = ?"
			if _, err := session.Exec(updateSQL, accessKeyId, accessKeySecret, "merchant", merchant.Id, "aws"); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return err
	}

	// 如果 server_ip 发生变化，统一进行联动（更新系统服务器gost转发；同步servers表host）
	if ipChanged {
		onMerchantServerIPChanged(merchant.Id, merchant.Port, merchant.ServerIP)
	}

	// 异步推送配置更新到商户服务
	AsyncNotifyConfigUpdate(merchant)

	return nil
}

// DeleteMerchant 删除商户
func DeleteMerchant(idStr string) (string, error) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return "", err
	}

	merchant, err := dbhelper.GetMerchantByID(id)
	if err != nil {
		return "", err
	}

	// 事务：删除商户及其关联的服务器和云账号
	if err := dbs.DBAdmin.WithTx(func(session *xorm.Session) error {
		// 1. 删除商户记录
		if _, err := session.ID(id).Delete(&entity.Merchants{}); err != nil {
			return fmt.Errorf("删除商户失败: %v", err)
		}

		// 2. 删除关联的商户服务器
		if _, err := session.Where("server_type = ? AND merchant_id = ?", 1, id).Delete(&entity.Servers{}); err != nil {
			return fmt.Errorf("删除商户服务器失败: %v", err)
		}

		// 3. 删除关联的云账号
		if _, err := session.Where("account_type = ? AND merchant_id = ?", "merchant", id).Delete(&entity.CloudAccounts{}); err != nil {
			return fmt.Errorf("删除云账号失败: %v", err)
		}

		return nil
	}); err != nil {
		return "", err
	}
	logx.Infof("删除商户 %s", merchant.Name)

	// 删除商户服务器上的本地转发（通过任务队列）
	if err := gostapi.EnqueueDeleteMerchantLocalForwards(merchant.ServerIP); err != nil {
		logx.Errorf("enqueue delete merchant local forwards task failed: %+v", err)
	}

	// 删除所有系统服务器上的 gost 转发服务（通过任务队列）
	enqueueDeleteGostServicesOnAllSystemServers(merchant.Port)
	return merchant.Name, nil
}

// enqueueDeleteGostServicesOnAllSystemServers 入队删除系统服务器上的商户转发任务
// 根据每个系统服务器的 forward_type 选择删除加密或直连转发
func enqueueDeleteGostServicesOnAllSystemServers(port int) {
	var sysServers []entity.Servers
	if err := dbs.DBAdmin.Where("server_type = ? AND status = 1", 2).Find(&sysServers); err != nil {
		logx.Errorf("list system servers err: %+v", err)
		return
	}

	encryptedCount := 0
	directCount := 0
	for _, s := range sysServers {
		var err error
		if s.ForwardType == entity.ForwardTypeDirect {
			// 直连转发
			err = gostapi.EnqueueDeleteMerchantDirectForwards(s.Host, port)
			directCount++
		} else {
			// 加密转发（默认）
			err = gostapi.EnqueueDeleteMerchantForwards(s.Host, port)
			encryptedCount++
		}
		if err != nil {
			logx.Errorf("enqueue delete merchant forwards task for server %d (%s, forward_type=%d) failed: %+v",
				s.Id, s.Host, s.ForwardType, err)
		}
	}
	logx.Infof("enqueued delete merchant forwards tasks for %d servers (encrypted: %d, direct: %d), port %d",
		len(sysServers), encryptedCount, directCount, port)
}

// onMerchantServerIPChanged 商户 server_ip 变更后的联动：
// 1) 同步 servers 表中该商户服务器记录的 host 为新 IP
// 2) 更新所有系统服务器上的 gost 转发目标（使用固定端口 10000/10001/10002）
func onMerchantServerIPChanged(merchantId int, port int, newServerIP string) {
	// 1) 同步 servers.host（商户服务器）
	_, err := dbs.DBAdmin.Table("servers").
		Where("server_type = ? AND merchant_id = ?", 1, merchantId).
		Update(map[string]any{
			"host":       newServerIP,
			"updated_at": time.Now(),
		})
	if err != nil {
		logx.Errorf("sync servers host failed for merchant %d: %+v", merchantId, err)
	}

	// 2) 查询商户 tunnelIP 并更新所有系统服务器上的 GOST 转发配置
	var merchant entity.Merchants
	tunnelIP := ""
	if has, err := dbs.DBAdmin.Where("id = ?", merchantId).Get(&merchant); err == nil && has {
		tunnelIP = merchant.TunnelIP
	}
	updateGostServicesOnSystemServers(merchantId, port, newServerIP, tunnelIP, 0)
}

// RefreshGostForwards 手动刷新商户的 GOST 转发配置（用于修复转发异常）
func RefreshGostForwards(merchantId int) error {
	var m entity.Merchants
	has, err := dbs.DBAdmin.Where("id = ?", merchantId).Get(&m)
	if err != nil {
		return fmt.Errorf("查询商户失败: %v", err)
	}
	if !has {
		return fmt.Errorf("商户不存在: %d", merchantId)
	}
	if m.ServerIP == "" || m.Port <= 0 {
		return fmt.Errorf("商户 server_ip 或 port 未配置")
	}
	onMerchantServerIPChanged(m.Id, m.Port, m.ServerIP)
	return nil
}

// GetMerchantByID 通过ID获取商户信息
func GetMerchantByID(id int) (*entity.Merchants, error) {
	return dbhelper.GetMerchantByID(id)
}

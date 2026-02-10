package dbhelper

import (
	"errors"
	"fmt"
	"server/pkg/dbs"
	"server/pkg/entity"
	"time"
)

// FindAllMerchants 获取所有商户
func FindAllMerchants() ([]entity.Merchants, error) {
	var merchants []entity.Merchants
	err := dbs.DBAdmin.Find(&merchants)
	if err != nil {
		return nil, err
	}
	return merchants, nil
}

func FindMerchantList(page, size int) ([]entity.Merchants, int64, error) {
	var merchants []entity.Merchants
	total, err := FindWithPagination(dbs.DBAdmin, "merchants", page, size, &merchants)
	if err != nil {
		return nil, 0, err
	}
	return merchants, total, nil
}

// 根据条件查询商户列表
func FindMerchantListWithCondition(page, size int, name, orderBy string, expiringSoon int, merchantNo string) ([]entity.Merchants, int64, error) {
	var merchants []entity.Merchants
	session := dbs.DBAdmin.Table("merchants")
	if name != "" {
		session = session.Where("name like ?", "%"+name+"%")
	}
	if merchantNo != "" {
		session = session.Where("no = ?", merchantNo)
	}
	switch expiringSoon {
	case 2: // 已过期
		session = session.Where("expired_at < ?", time.Now())
	case 1: // 即将过期
		session = session.Where("expired_at < ? AND expired_at > ?", time.Now().AddDate(0, 0, 3), time.Now())
	}
	session.OrderBy(orderBy)
	total, err := session.Limit(size, (page-1)*size).FindAndCount(&merchants)
	if err != nil {
		return nil, 0, err
	}
	return merchants, total, nil
}

func GetMerchantByID(id int) (*entity.Merchants, error) {
	var merchant entity.Merchants
	ok, err := dbs.DBAdmin.ID(id).Get(&merchant)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("merchant not found %d", id)
	}
	return &merchant, nil
}
func GetMerchantByServerID(serverID int) (*entity.Merchants, error) {
	var server entity.Servers
	ok, err := dbs.DBAdmin.ID(serverID).Get(&server)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("server not found %d", serverID)
	}
	var merchant entity.Merchants
	ok, err = dbs.DBAdmin.ID(server.MerchantId).Get(&merchant)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("merchant not found %d", server.MerchantId)
	}
	return &merchant, nil
}

func GetMerchantByTable(table string) (*entity.Merchants, error) {
	var merchant entity.Merchants
	ok, err := dbs.DBAdmin.Where("`table` = ?", table).Get(&merchant)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("merchant not found")
	}
	return &merchant, nil
}

// GetMerchantByUid 根据uid获取商户信息
func GetMerchantByUid(uid string) (*entity.Merchants, error) {
	var merchant entity.Merchants
	ok, err := dbs.DBAdmin.Where("uuid = ?", uid).Get(&merchant)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("merchant not found")
	}
	return &merchant, nil
}

// GetMerchantByNo 根据商户编号获取商户信息
func GetMerchantByNo(merchantNo string) (*entity.Merchants, error) {
	var merchant entity.Merchants
	ok, err := dbs.DBAdmin.Where("no = ?", merchantNo).Get(&merchant)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("商户不存在: %s", merchantNo)
	}
	return &merchant, nil
}

// GetMerchantByServerIP 根据服务器IP获取商户信息
func GetMerchantByServerIP(serverIP string) (*entity.Merchants, error) {
	var merchant entity.Merchants
	ok, err := dbs.DBAdmin.Where("server_ip = ?", serverIP).Get(&merchant)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("未找到IP对应的商户: %s", serverIP)
	}
	return &merchant, nil
}

// CreateMerchant 创建商户
func CreateMerchant(merchant *entity.Merchants) error {
	now := time.Now()
	merchant.CreatedAt = now
	merchant.UpdatedAt = now
	_, err := dbs.DBAdmin.Insert(merchant)
	return err
}

// DeleteMerchant 删除商户
func DeleteMerchant(id int) error {
	_, err := dbs.DBAdmin.ID(id).Delete(new(entity.Merchants))
	return err
}

// UpdateMerchant 更新商户
func UpdateMerchant(merchant *entity.Merchants) error {
	now := time.Now()
	merchant.UpdatedAt = now
	_, err := dbs.DBAdmin.ID(merchant.Id).Update(merchant)
	return err
}

// CheckMerchantPortInUse 检查是否存在占用指定端口的商户
func CheckMerchantPortInUse(port int) (bool, error) {
	return dbs.DBAdmin.Where("port = ?", port).Exist(&entity.Merchants{})
}

// CheckMerchantPortInUseByOther 检查是否被其他商户占用指定端口
func CheckMerchantPortInUseByOther(port int, excludeId int) (bool, error) {
	return dbs.DBAdmin.Where("port = ? AND id != ?", port, excludeId).Exist(&entity.Merchants{})
}

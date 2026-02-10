package dbhelper

import (
	"fmt"
	"server/pkg/dbs"
	"server/pkg/entity"
)

// FindWithPagination 通用分页查询函数
// tableName: 表名
// page: 页码
// size: 每页大小
// result: 结果切片的指针
func FindWithPagination(db *dbs.XormDB, tableName string, page, size int, result interface{}) (int64, error) {
	total, err := db.Table(tableName).Limit(size, (page-1)*size).FindAndCount(result)
	if err != nil {
		return 0, err
	}
	return total, nil
}

// FindWithPaginationAndCondition 通用分页查询函数w，支持额外的查询条件
// tableName: 表名
// page: 页码
// size: 每页大小
// result: 结果切片的指针
// conditions: 查询条件，可以是字符串或者 map[string]interface{}
// args: 查询条件的参数
func FindWithPaginationAndCondition(db *dbs.XormDB, tableName string, page, size int, result interface{}, conditions interface{}, args ...interface{}) (int64, error) {
	// 构建查询
	query := db.Table(tableName)

	// 添加条件
	if conditions != nil {
		query = query.Where(conditions, args...)
	}

	// 执行分页查询
	err := query.Limit(size, (page-1)*size).Find(result)
	if err != nil {
		return 0, err
	}

	// 计算总记录数（带条件）
	countQuery := db.Table(tableName)
	if conditions != nil {
		countQuery = countQuery.Where(conditions, args...)
	}

	var total int64
	total, err = countQuery.Count()
	if err != nil {
		return 0, err
	}

	return total, nil
}

// FindWithPaginationAndMultiConditions 通用分页查询函数，支持多个查询条件
// tableName: 表名
// page: 页码
// size: 每页大小
// result: 结果切片的指针
// condMap: 条件映射，键为字段名，值为查询值
// likeFields: 需要使用 LIKE 查询的字段列表
// orderBy: 排序字段，格式为 "field ASC" 或 "field DESC"
func FindWithPaginationAndMultiConditions(
	db *dbs.XormDB,
	tableName string,
	page, size int,
	result interface{},
	condMap map[string]interface{},
	likeFields []string,
	orderBy string,
) (int64, error) {
	// 构建查询
	query := db.Table(tableName)

	// 添加条件
	if len(condMap) > 0 {
		for field, value := range condMap {
			if value != nil && value != "" {
				// 检查是否为 LIKE 查询字段
				isLikeField := false
				for _, likeField := range likeFields {
					if field == likeField {
						isLikeField = true
						break
					}
				}

				if isLikeField {
					// LIKE 查询
					query = query.Where(field+" LIKE ?", "%"+value.(string)+"%")
				} else {
					// 精确匹配
					query = query.Where(field+" = ?", value)
				}
			}
		}
	}

	// 添加排序
	if orderBy != "" {
		query = query.OrderBy(orderBy)
	}

	// 执行查询
	total, err := query.Limit(size, (page-1)*size).FindAndCount(result)
	if err != nil {
		return 0, err
	}
	return total, nil
}

// GetCloudAccountByID 根据ID获取云账号信息
func GetCloudAccountByID(id int64) (*entity.CloudAccounts, error) {
	var account entity.CloudAccounts
	ok, err := dbs.DBAdmin.ID(id).Get(&account)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("cloud account not found %d", id)
	}
	return &account, nil
}

// GetCloudAccountByMerchantType 获取某商户下指定云类型的启用账号（优先返回最新创建）
func GetCloudAccountByMerchantType(merchantId int, cloudType string) (*entity.CloudAccounts, error) {
	var account entity.CloudAccounts
	ok, err := dbs.DBAdmin.Where("account_type = ? AND merchant_id = ? AND cloud_type = ? AND status = 1 AND access_key_id != '' AND access_key_secret != ''", "merchant", merchantId, cloudType).
		Desc("id").
		Get(&account)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("merchant %d cloud account (%s) not found", merchantId, cloudType)
	}
	return &account, nil
}

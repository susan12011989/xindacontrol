package resource_group

import (
	"fmt"
	"server/internal/server/model"
	"server/pkg/dbs"
	"server/pkg/entity"
	"time"
)

// ListGroups 获取指定资源类型的分组列表
func ListGroups(resourceType string) ([]model.ResourceGroupResp, error) {
	var groups []entity.ResourceGroups
	err := dbs.DBAdmin.Where("resource_type = ?", resourceType).
		OrderBy("sort_order ASC, id ASC").Find(&groups)
	if err != nil {
		return nil, err
	}

	result := make([]model.ResourceGroupResp, len(groups))
	for i, g := range groups {
		count := 0
		switch resourceType {
		case entity.ResourceGroupTypeIpEmbedTarget:
			cnt, _ := dbs.DBAdmin.Where("group_id = ?", g.Id).Count(&entity.IpEmbedTargets{})
			count = int(cnt)
		case entity.ResourceGroupTypeServer:
			cnt, _ := dbs.DBAdmin.Where("group_id = ?", g.Id).Count(&entity.Servers{})
			count = int(cnt)
		}

		result[i] = model.ResourceGroupResp{
			Id:           g.Id,
			Name:         g.Name,
			ResourceType: g.ResourceType,
			SortOrder:    g.SortOrder,
			Count:        count,
			CreatedAt:    g.CreatedAt.Format(time.DateTime),
		}
	}
	return result, nil
}

// CreateGroup 创建分组
func CreateGroup(resourceType string, req model.ResourceGroupReq) (int, error) {
	group := &entity.ResourceGroups{
		Name:         req.Name,
		ResourceType: resourceType,
		SortOrder:    req.SortOrder,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	_, err := dbs.DBAdmin.Insert(group)
	if err != nil {
		return 0, fmt.Errorf("创建分组失败: %v", err)
	}
	return group.Id, nil
}

// UpdateGroup 更新分组
func UpdateGroup(id int, req model.ResourceGroupReq) error {
	_, err := dbs.DBAdmin.Table("resource_groups").Where("id = ?", id).Update(map[string]interface{}{
		"name":       req.Name,
		"sort_order": req.SortOrder,
		"updated_at": time.Now(),
	})
	return err
}

// DeleteGroup 删除分组并重置关联资源的 group_id
func DeleteGroup(id int, resourceType string) error {
	switch resourceType {
	case entity.ResourceGroupTypeIpEmbedTarget:
		_, _ = dbs.DBAdmin.Table("ip_embed_targets").Where("group_id = ?", id).
			Update(map[string]interface{}{"group_id": 0})
	case entity.ResourceGroupTypeServer:
		_, _ = dbs.DBAdmin.Table("servers").Where("group_id = ?", id).
			Update(map[string]interface{}{"group_id": 0})
	}
	_, err := dbs.DBAdmin.ID(id).Delete(&entity.ResourceGroups{})
	return err
}

package global

import (
	"errors"
	"server/internal/server/model"
	"server/pkg/dbs"
	"server/pkg/entity"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// QueryOssUrl 查询OSS URL列表
func QueryOssUrl(req model.QueryOssUrlReq) (model.QueryOssUrlResponse, error) {
	var resp model.QueryOssUrlResponse

	session := dbs.DBAdmin.Table("global_oss_url")

	// 条件过滤
	if req.Url != "" {
		session = session.Where("url LIKE ?", "%"+req.Url+"%")
	}

	// 使用 FindAndCount 一次性完成计数和查询
	var ossUrls []entity.GlobalOssUrl
	offset := (req.Page - 1) * req.Size
	total, err := session.Desc("id").Limit(req.Size, offset).FindAndCount(&ossUrls)
	if err != nil {
		logx.Errorf("query oss url err: %+v", err)
		return resp, err
	}
	resp.Total = int(total)

	// 转换为响应格式
	for _, item := range ossUrls {
		resp.List = append(resp.List, model.OssUrlResp{
			Id:        item.Id,
			Url:       item.Url,
			UpdatedAt: item.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return resp, nil
}

// CreateOssUrl 创建OSS URL
func CreateOssUrl(req model.CreateOssUrlReq) (int64, error) {
	ossUrl := entity.GlobalOssUrl{
		Url:       req.Url,
		UpdatedAt: time.Now(),
	}

	affected, err := dbs.DBAdmin.Insert(&ossUrl)
	if err != nil {
		logx.Errorf("create oss url err: %+v", err)
		return 0, err
	}
	if affected == 0 {
		return 0, errors.New("创建失败")
	}
	return int64(ossUrl.Id), nil
}

// UpdateOssUrl 更新OSS URL
func UpdateOssUrl(id int64, req model.UpdateOssUrlReq) error {
	updates := map[string]interface{}{
		"url": req.Url,
	}

	affected, err := dbs.DBAdmin.Table("global_oss_url").
		Where("id = ?", id).
		Update(updates)
	if err != nil {
		logx.Errorf("update oss url err: %+v", err)
		return err
	}
	if affected == 0 {
		return errors.New("记录不存在")
	}
	return nil
}

// DeleteOssUrl 删除OSS URL
func DeleteOssUrl(id int64) error {
	affected, err := dbs.DBAdmin.Where("id = ?", id).Delete(&entity.GlobalOssUrl{})
	if err != nil {
		logx.Errorf("delete oss url err: %+v", err)
		return err
	}
	if affected == 0 {
		return errors.New("记录不存在")
	}
	return nil
}

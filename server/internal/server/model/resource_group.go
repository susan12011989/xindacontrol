package model

// ========== 资源分组 ==========

// ResourceGroupReq 创建/更新分组请求
type ResourceGroupReq struct {
	Name      string `json:"name" binding:"required"`
	SortOrder int    `json:"sort_order"`
}

// ResourceGroupResp 分组响应
type ResourceGroupResp struct {
	Id           int    `json:"id"`
	Name         string `json:"name"`
	ResourceType string `json:"resource_type"`
	SortOrder    int    `json:"sort_order"`
	Count        int    `json:"count"`
	CreatedAt    string `json:"created_at"`
}

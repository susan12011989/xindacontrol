package model

// ========== 全局管理 ==========

// 查询OSS URL请求
type QueryOssUrlReq struct {
	Pagination
	Url string `json:"url" form:"url"` // URL（模糊查询）
}

// 创建OSS URL请求
type CreateOssUrlReq struct {
	Url string `json:"url" binding:"required"` // URL
}

// 更新OSS URL请求
type UpdateOssUrlReq struct {
	Url string `json:"url" binding:"required"` // URL
}

// OSS URL响应
type OssUrlResp struct {
	Id        int    `json:"id"`
	Url       string `json:"url"`
	UpdatedAt string `json:"updated_at"`
}

// OSS URL列表响应
type QueryOssUrlResponse struct {
	List  []OssUrlResp `json:"list"`
	Total int          `json:"total"`
}

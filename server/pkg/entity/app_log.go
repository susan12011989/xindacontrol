package entity

// AppLogListItem 应用日志列表项
// 匹配商户端 TangSengDaoDaoServer/modules/control/db.go 的 AppLogListItem
type AppLogListItem struct {
	ID         int64  `json:"id"`
	UID        string `json:"uid"`
	LogType    string `json:"log_type"`
	LogDate    string `json:"log_date"`
	FileName   string `json:"file_name"`
	MinioPath  string `json:"minio_path"`
	FileSize   int64  `json:"file_size"`
	DeviceInfo string `json:"device_info"`
	AppVersion string `json:"app_version"`
	CreatedAt  string `json:"created_at"`
	Name       string `json:"name"`
	Phone      string `json:"phone"`
	ShortNo    string `json:"short_no"`
}

// AppLogQueryResp 应用日志查询响应
// 匹配商户端 TangSengDaoDaoServer/modules/control/db.go 的 AppLogQueryResp
type AppLogQueryResp struct {
	List  []AppLogListItem `json:"list"`
	Total int64            `json:"total"`
	Err   string           `json:"err,omitempty"` // 用于内部错误传递
}

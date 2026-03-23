package model

// ========== WuKongIM 监控 ==========

// WuKongIMBaseReq 基础请求（需要指定服务器）
type WuKongIMBaseReq struct {
	ServerId int `json:"server_id" form:"server_id" binding:"required"`
}

// WuKongIMVarzResp 系统变量响应（来自 WuKongIM /varz）
type WuKongIMVarzResp struct {
	ServerID            string `json:"server_id"`
	ServerName          string `json:"server_name"`
	Version             string `json:"version"`
	Connections         int    `json:"connections"`
	UserHandlerCount    int    `json:"user_handler_count"`
	UserHandlerConnCount int   `json:"user_handler_conn_count"`
	Uptime              string `json:"uptime"`
	Goroutine           int    `json:"goroutine"`
	Mem                 int64  `json:"mem"`
	CPU                 any    `json:"cpu"`
	InMsgs              int64  `json:"in_msgs"`
	OutMsgs             int64  `json:"out_msgs"`
	InBytes             int64  `json:"in_bytes"`
	OutBytes            int64  `json:"out_bytes"`
	SlowClients         int64  `json:"slow_clients"`
	RetryQueue          int64  `json:"retry_queue"`
	TCPAddr             string `json:"tcp_addr"`
	WSAddr              string `json:"ws_addr"`
	WSSAddr             string `json:"wss_addr"`
	ManagerAddr         string `json:"manager_addr"`
	ManagerOn           int    `json:"manager_on"`
	Commit              string `json:"commit"`
	CommitDate          string `json:"commit_date"`
	TreeState           string `json:"tree_state"`
	APIURL              string `json:"api_url"`
	ManagerUID          string `json:"manager_uid"`
	ManagerTokenOn      int    `json:"manager_token_on"`
	Conns               any    `json:"conns,omitempty"`
}

// WuKongIMConnzReq 连接查询请求
type WuKongIMConnzReq struct {
	ServerId int    `json:"server_id" form:"server_id" binding:"required"`
	Offset   int    `json:"offset" form:"offset"`
	Limit    int    `json:"limit" form:"limit"`
	UID      string `json:"uid" form:"uid"`
	Sort     string `json:"sort" form:"sort"`
}

// WuKongIMConnInfo 单个连接信息
type WuKongIMConnInfo struct {
	ID              int64  `json:"id"`
	UID             string `json:"uid"`
	IP              string `json:"ip"`
	Port            int    `json:"port"`
	LastActivity    string `json:"last_activity"`
	Uptime          string `json:"uptime"`
	Idle            string `json:"idle"`
	PendingBytes    int    `json:"pending_bytes"`
	InMsgs          int64  `json:"in_msgs"`
	OutMsgs         int64  `json:"out_msgs"`
	InMsgBytes      int64  `json:"in_msg_bytes"`
	OutMsgBytes     int64  `json:"out_msg_bytes"`
	InPackets       int64  `json:"in_packets"`
	OutPackets      int64  `json:"out_packets"`
	InPacketBytes   int64  `json:"in_packet_bytes"`
	OutPacketBytes  int64  `json:"out_packet_bytes"`
	Device          string `json:"device"`
	DeviceID        string `json:"device_id"`
	Version         int    `json:"version"`
	ProxyTypeFormat string `json:"proxy_type_format"`
	LeaderID        int64  `json:"leader_id"`
	NodeID          int64  `json:"node_id"`
}

// WuKongIMConnzResp 连接列表响应（来自 WuKongIM /connz）
type WuKongIMConnzResp struct {
	Connections []WuKongIMConnInfo `json:"connections"`
	Now         string             `json:"now"`
	Total       int                `json:"total"`
	Offset      int                `json:"offset"`
	Limit       int                `json:"limit"`
}

// WuKongIMOnlineStatusReq 用户在线状态查询请求
type WuKongIMOnlineStatusReq struct {
	ServerId int      `json:"server_id" binding:"required"`
	UIDs     []string `json:"uids" binding:"required"`
}

// WuKongIMOnlineStatusItem 用户在线状态
type WuKongIMOnlineStatusItem struct {
	UID        string `json:"uid"`
	DeviceFlag int    `json:"device_flag"`
	Online     int    `json:"online"`
}

// WuKongIMDeviceQuitReq 强制下线请求
type WuKongIMDeviceQuitReq struct {
	ServerId   int    `json:"server_id" binding:"required"`
	UID        string `json:"uid" binding:"required"`
	DeviceFlag int    `json:"device_flag"` // -1=全部, 0=APP, 1=WEB, 2=PC
}

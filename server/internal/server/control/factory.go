package control

import (
	"fmt"
	"sync"

	"github.com/zeromicro/go-zero/core/logx"
)

var (
	// 全局控制器实例
	globalController IController
	globalGost       IGostController
	globalTunnel     ITunnelController
	controllerOnce   sync.Once
)

// Config 控制器配置
type Config struct {
	Mode Mode `json:"mode" yaml:"Mode"` // local 或 cluster
}

// Init 根据配置初始化全局控制器（应在服务启动时调用一次）
func Init(cfg Config) {
	controllerOnce.Do(func() {
		switch cfg.Mode {
		case ModeLocal:
			local := NewLocalController()
			globalController = local
			globalGost = newLocalGostController(local.executor)
			globalTunnel = newLocalTunnelController(local.executor)
			logx.Info("[Control] 初始化单机模式控制器")
		case ModeCluster:
			cluster := NewClusterController()
			globalController = cluster
			globalGost = newClusterGostController(cluster)
			globalTunnel = newClusterTunnelController(cluster)
			logx.Info("[Control] 初始化多机模式控制器")
		default:
			// 默认多机模式（兼容现有行为）
			cluster := NewClusterController()
			globalController = cluster
			globalGost = newClusterGostController(cluster)
			globalTunnel = newClusterTunnelController(cluster)
			logx.Infof("[Control] 未指定模式，默认使用多机模式控制器 (mode=%s)", cfg.Mode)
		}
	})
}

// Get 获取全局控制器
func Get() IController {
	if globalController == nil {
		panic("control: 控制器未初始化，请先调用 control.Init()")
	}
	return globalController
}

// GetCluster 获取多机模式控制器（仅多机模式可用，提供带 serverId 的扩展方法）
func GetCluster() (*ClusterController, error) {
	ctrl := Get()
	if ctrl.Mode() != ModeCluster {
		return nil, fmt.Errorf("当前为单机模式，不支持多机操作")
	}
	return ctrl.(*ClusterController), nil
}

// GetLocal 获取单机模式控制器
func GetLocal() (*LocalController, error) {
	ctrl := Get()
	if ctrl.Mode() != ModeLocal {
		return nil, fmt.Errorf("当前为多机模式，不支持单机操作")
	}
	return ctrl.(*LocalController), nil
}

// IsLocalMode 判断是否为单机模式
func IsLocalMode() bool {
	return Get().Mode() == ModeLocal
}

// IsClusterMode 判断是否为多机模式
func IsClusterMode() bool {
	return Get().Mode() == ModeCluster
}

// GetGost 获取 GOST 控制器
func GetGost() IGostController {
	if globalGost == nil {
		panic("control: GOST 控制器未初始化")
	}
	return globalGost
}

// GetGostCluster 获取多机模式 GOST 控制器（提供带 serverId 的方法）
func GetGostCluster() (*clusterGostController, error) {
	if !IsClusterMode() {
		return nil, fmt.Errorf("当前为单机模式，不支持多机 GOST 操作")
	}
	return globalGost.(*clusterGostController), nil
}

// GetTunnel 获取隧道控制器
func GetTunnel() ITunnelController {
	if globalTunnel == nil {
		panic("control: 隧道控制器未初始化")
	}
	return globalTunnel
}

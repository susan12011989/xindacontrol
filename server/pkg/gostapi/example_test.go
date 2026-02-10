package gostapi_test

import (
	"fmt"
	"log"

	"server/pkg/gostapi"
)

// 这个文件展示了如何使用 GOST API 客户端

// Example_simple 展示最简单的使用方式：直接使用包级别函数
func Example_simple() {
	// GOST 服务器的 IP 地址
	ip := "127.0.0.1"

	// 1. 获取当前配置
	config, err := gostapi.GetConfig(ip, "json")
	if err != nil {
		log.Printf("获取配置失败: %v", err)
		return
	}
	fmt.Printf("当前配置: %+v\n", config)

	// 2. 保存配置到文件
	resp, err := gostapi.SaveConfig(ip, "yaml", "")
	if err != nil {
		log.Printf("保存配置失败: %v", err)
		return
	}
	fmt.Printf("保存配置响应: %+v\n", resp)

	// 3. 获取服务列表
	serviceList, err := gostapi.GetServiceList(ip)
	if err != nil {
		log.Printf("获取服务列表失败: %v", err)
		return
	}
	fmt.Printf("服务数量: %d\n", serviceList.Count)
	for _, svc := range serviceList.List {
		fmt.Printf("服务: %s, 地址: %s\n", svc.Name, svc.Addr)
	}

	// 4. 获取服务详情
	service, err := gostapi.GetService(ip, "service-0")
	if err != nil {
		log.Printf("获取服务详情失败: %v", err)
		return
	}
	fmt.Printf("服务详情: %+v\n", service)

	// 5. 创建新服务
	newService := &gostapi.ServiceConfig{
		Name: "service-0",
		Addr: ":8080",
		Handler: &gostapi.HandlerConfig{
			Type: "http",
		},
		Listener: &gostapi.ListenerConfig{
			Type: "tcp",
		},
	}
	resp, err = gostapi.CreateService(ip, newService)
	if err != nil {
		log.Printf("创建服务失败: %v", err)
		return
	}
	fmt.Printf("创建服务响应: %+v\n", resp)

	// 6. 更新服务
	updatedService := &gostapi.ServiceConfig{
		Name: "service-0",
		Addr: ":8080",
		Handler: &gostapi.HandlerConfig{
			Type: "socks5",
		},
		Listener: &gostapi.ListenerConfig{
			Type: "tcp",
		},
	}
	resp, err = gostapi.UpdateService(ip, "service-0", updatedService)
	if err != nil {
		log.Printf("更新服务失败: %v", err)
		return
	}
	fmt.Printf("更新服务响应: %+v\n", resp)

	// 7. 删除服务
	resp, err = gostapi.DeleteService(ip, "service-0")
	if err != nil {
		log.Printf("删除服务失败: %v", err)
		return
	}
	fmt.Printf("删除服务响应: %+v\n", resp)
}

// Example_withClient 展示使用自定义客户端的方式（如果需要自定义配置）
func Example_withClient() {
	// 创建客户端（如果需要自定义超时等配置）
	client := gostapi.NewClient()

	// GOST 服务器的 IP 地址
	ip := "127.0.0.1"

	// 使用客户端方法
	serviceList, err := client.GetServiceList(ip)
	if err != nil {
		log.Printf("获取服务列表失败: %v", err)
		return
	}
	fmt.Printf("服务数量: %d\n", serviceList.Count)
}

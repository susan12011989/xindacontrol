package main

import (
	"fmt"
	"log"
	"server/pkg/gostapi"
)

func main() {
	// createService()
	queryService()
	// queryChain()
}

func createService() {
	serviceName, err := gostapi.CreateRelayTLSForward("47.112.9.162", 11115, "16.162.88.246", 10544)
	if err != nil {
		log.Fatalf("创建服务失败: %v", err)
	}
	fmt.Println("服务创建成功: ", serviceName)
}

func deleteService() {
	err := gostapi.DeleteRelayTLSForward("47.112.9.162", 11115)
	if err != nil {
		log.Fatalf("删除服务失败: %v", err)
	}
	fmt.Println("服务删除成功")
}

func queryService() {
	services, err := gostapi.GetServiceList("47.112.9.162")
	if err != nil {
		log.Fatalf("查询服务失败: %v", err)
	}
	fmt.Println("服务查询成功: ", services)
}

func queryChain() {
	chains, err := gostapi.GetChainList("47.112.9.162")
	if err != nil {
		log.Fatalf("查询链失败: %v", err)
	}
	fmt.Println("链查询成功: ", chains)
}

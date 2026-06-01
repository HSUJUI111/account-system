package main

import (
	"account-system/config"
	"account-system/internal/api"
	"account-system/internal/repository"
	"account-system/pkg"
	"log"
)

func main() {
	cfg, err := config.Load("../config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	log.Printf("配置加载成功: %s", cfg)
	repository.InitDB(cfg.Mysql)
	pkg.Init()
	r := api.SetupRouter()
	err = r.Run(":8080")
	if err != nil {
		log.Fatalf("启动服务器失败: %v", err)
	}
}

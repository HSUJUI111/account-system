package main

import (
	"account-system/config"
	"account-system/internal/handler"
	"account-system/internal/repository"
	"account-system/internal/service"
	"account-system/pkg/idgen"
	"log"
)

func main() {
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %+v", err)
	}
	log.Printf("配置加载成功: %+v", cfg)
	repository.InitRedis(cfg.Redis)
	service.InitAuth(cfg.Jwt)
	repository.InitDB(cfg.Mysql)
	idgen.Init()
	r := handler.SetupRouter()
	err = r.Run(":8080")
	if err != nil {
		log.Fatalf("启动服务器失败: %+v", err)
	}
}

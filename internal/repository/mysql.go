package repository

import (
	"account-system/config"
	"account-system/internal/model"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB(cfg config.MysqlConfig) {
	db, err := gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{
		TranslateError: true,
	})

	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)

	err = db.AutoMigrate(model.Account{}, model.DepositOrder{}, model.AccountTransaction{}, &model.TransferOrder{}, model.WithdrawOrder{}, model.ReconcileAlert{}, model.User{})
	if err != nil {
		log.Fatalf("自动建表失败: %v", err)
	}
	DB = db
	log.Println("数据库连接成功")
}

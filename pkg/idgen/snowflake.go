package idgen

import (
	"log"

	"github.com/bwmarrin/snowflake"
)

var node *snowflake.Node

func Init() {
	n, err := snowflake.NewNode(1)
	if err != nil {
		log.Fatal("雪花初始化失败", err)
	}
	node = n
}

func GenerateNo(prefix string) string {
	return prefix + node.Generate().String()
}

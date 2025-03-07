package main

import (
	"go-im/cmd/server"
	"log"
)

func main() {
	// 1.初始化日志模块
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	// 2.创建服务器
	imServer := im_server.NewServer("0.0.0.0", 8888)
	// 3.启动服务器
	imServer.Start()
}

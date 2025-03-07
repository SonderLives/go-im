package im_server

import (
	"go-im/pkg/utils"
	"log"
	"net"
)

type User struct {
	UserName       string
	Address        net.Addr
	Connection     net.Conn
	MessageChannel chan string
}

func NewUserConnect(conn net.Conn) *User {
	user := &User{
		UserName:       conn.RemoteAddr().String(),
		Address:        conn.RemoteAddr(),
		Connection:     conn,
		MessageChannel: make(chan string),
	}
	// 监听消息通道
	go user.ListenMessageChannel()

	return user
}

func (user *User) ListenMessageChannel() {
	for {
		message, ok := <-user.MessageChannel
		if !ok {
			// 通道已经关闭，退出循环
			log.Println(utils.Red("[-] 用户消息通道已关闭，停止监听"))
			break
		}

		_, err := user.Connection.Write([]byte(message))
		if err != nil {
			log.Println(utils.Red("[-] 向客户端发送数据时出现错误, ", err.Error()))
		}
	}
}

func (user *User) Offline() {
	err := user.Connection.Close()
	close(user.MessageChannel)
	if err != nil {
		log.Println(utils.Red("[-] 关闭连接时出现错误,", err.Error()))
	}
}

func (user *User) Rename(name string) {
	user.UserName = name
}

package im_server

import (
	"fmt"
	"go-im/pkg/utils"
	"io"
	"log"
	"net"
	"strings"
	"sync"
)

type Server struct {
	Ip             string
	Port           int
	OnlineMap      sync.Map
	MessageChannel chan string // 服务器的消息存储队列
}

func NewServer(ip string, port int) *Server {
	server := &Server{
		Ip:             ip,
		Port:           port,
		OnlineMap:      sync.Map{},
		MessageChannel: make(chan string),
	}
	return server
}

func (server *Server) Start() {
	// 1. 创建套接字并进行监听
	formatString := fmt.Sprintf("%s:%d", server.Ip, server.Port)
	log.Println(utils.Blue("[*] 服务器开始Listen ", formatString), "😎")

	listener, err := net.Listen("tcp", formatString)
	if err != nil {
		log.Fatalln(utils.Red("[-] TCP 服务器监听失败, ", err.Error()), "😭")
	}
	log.Println(utils.Green("[+] 服务器Listen成功! "), "😆")

	// 2.服务退出以后关闭服务器
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
			log.Fatalln(utils.Red("[-] 监听器listener关闭失败, ", err.Error()), "😬")
		}
	}(listener)

	// 3.监听服务器的消息队列，准备进行消息分发
	go server.ListenMessageChannel()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println(utils.Blue("[-] 收到新的TCP连接, 但是Accept失败了", conn.RemoteAddr(), err.Error()))
		}
		log.Println(utils.Blue("[*] 收到新的TCP连接：", conn.RemoteAddr()), "🥰")
		// 3.处理业务逻辑
		go server.HandleConnect(conn)
	}
}

func (server *Server) HandleConnect(conn net.Conn) {
	// 1. 创建用户并将用户加入用户在线表中
	user := NewUserConnect(conn)
	server.OnlineMap.Store(user.UserName, user)

	// 2.广播上线消息
	notifyMessage := fmt.Sprint("已上线~ 😘")
	formatMessage := fmt.Sprintf("%s> %s\n", user.UserName, notifyMessage)
	server.BroadcastOfficialNotify(formatMessage)

	// 3.处理用户消息
	go server.HandleMessage(user)
}

func (server *Server) ListenMessageChannel() {
	for {
		message := <-server.MessageChannel
		server.Broadcast(message)
	}
}

func (server *Server) Broadcast(message string) {
	// 进行广播消息
	server.OnlineMap.Range(func(key, value interface{}) bool {
		if user, ok := value.(*User); ok {
			user.MessageChannel <- message
		} else {
			log.Panicln(utils.Red("[-] 类型断言失败 value is not *User"))
		}
		return true
	})
}

func (server *Server) BroadcastOfficialNotify(message string) {
	officialNotifyMessage := fmt.Sprintf("%s %s\n", "[official notify]", message)
	server.MessageChannel <- officialNotifyMessage
}

func (server *Server) HandleMessage(user *User) {
	buf := make([]byte, 4096)
	for {
		n, err := user.Connection.Read(buf)

		if n == 0 || err == io.EOF {
			// 用户下线,关闭连接
			server.OnlineMap.Delete(user.UserName)
			formatMessage := fmt.Sprintf("%s %s> %s\n", "[official notify]", user.UserName, "下线了 😣")
			server.MessageChannel <- formatMessage
			user.Offline()
			break
		}

		if err != nil {
			log.Println(utils.Red("[-] 读取用户输入时出现错误,", err.Error()))
			continue
		}

		// 命令解析
		message := string(buf[:n-1])
		if strings.HasPrefix(message, "shell") {
			commandStr := strings.TrimPrefix(message, "shell")
			parts := strings.Fields(commandStr)
			server.CommandExec(user, parts)
		} else {
			// 将消息推送服务器的消息存储队列中
			formatMessage := fmt.Sprintf("%s> %s\n", user.UserName, message)
			server.MessageChannel <- formatMessage
		}
	}
}

func (server *Server) CommandExec(user *User, command []string) {
	switch command[0] {
	case "online":
		// 返回当前的在线用户列表
		server.HandleOnlineCommand(user)
	case "rename":
		// 修改用户昵称
		if len(command) < 2 || len(command[1]) < 1 {
			SendCommandError(user)
		} else {
			server.HandleRenameCommand(user, command[1])
		}
	default:
		SendCommandError(user)
	}
}

func (server *Server) HandleOnlineCommand(user *User) {
	// 存储在线用户列表
	var onlineUsers []string

	server.OnlineMap.Range(func(key, value interface{}) bool {
		if user, ok := value.(*User); ok {
			onlineUsers = append(onlineUsers, user.UserName)
		} else {
			log.Panicln(utils.Red("[-] 类型断言失败 value is not *User"))
		}
		return true
	})

	// 格式化在线用户列表
	var response string
	if len(onlineUsers) == 0 {
		response = "[official notify] 当前没有在线用户。"
	} else {
		response = "[official notify] 当前在线用户列表：\n"
		for _, user := range onlineUsers {
			response += fmt.Sprintf("- %s\n", user)
		}
	}

	// 向用户发送消息
	SendMessage(user, response)
}

func SendMessage(user *User, message string) {
	if len(message) != 0 {
		user.MessageChannel <- message
	}
}

func SendCommandError(user *User) {
	tipString := `[!] 你的输入有误, 示例： 
- shell online 
- shell rename nickname
`
	SendMessage(user, tipString)
}

func (server *Server) HandleRenameCommand(user *User, name string) {
	// 1.判断当前用户名是否存在
	_, ok := server.OnlineMap.Load(name)
	if ok {
		// 1.1 存在则返回错误
		SendMessage(user, "[!] 用户名已存在\n")
	} else {
		// 1.2 不存在即可修改 user和OnlineMap
		oldName := user.UserName
		user.Rename(name)
		server.OnlineMap.Delete(oldName)
		server.OnlineMap.Store(user.UserName, user)
		// 系统广播改名消息
		notifyMessage := fmt.Sprintf("%s%s\n", oldName, fmt.Sprintf("将昵称修改为 %s", user.UserName))
		server.BroadcastOfficialNotify(notifyMessage)
	}
}

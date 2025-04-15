package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// Message 定义消息结构，新增 Role 字段用于标识身份（B 或 C），以及注册消息
type Message struct {
	Type      string `json:"type"`
	SDP       string `json:"sdp,omitempty"`
	Candidate string `json:"candidate,omitempty"`
	Role      string `json:"role,omitempty"` // 用于注册时标识身份，如 "B"
	From      string `json:"from,omitempty"` // 用于标识消息来源
}

// 全局连接存储及角色映射
var clients = make(map[*websocket.Conn]bool)
var roles = make(map[*websocket.Conn]string)
var mu sync.Mutex

// 专门存放注册为 B 的连接
var bConn *websocket.Conn

// 处理每个 websocket 连接
func handleWebSocket(conn *websocket.Conn) {
	defer func() {
		mu.Lock()
		delete(clients, conn)
		delete(roles, conn)
		// 如果断开的是 B 连接，清空 bConn
		if bConn == conn {
			bConn = nil
		}
		mu.Unlock()
		conn.Close()
	}()

	// 新连接加入
	mu.Lock()
	clients[conn] = true
	mu.Unlock()

	for {
		var msg Message
		if err := conn.ReadJSON(&msg); err != nil {
			log.Println("Error reading JSON:", err)
			break
		}

		// 处理注册消息，客户端在连接后应先发送注册消息
		if msg.Type == "register" {
			mu.Lock()
			roles[conn] = msg.Role
			if msg.Role == "B" {
				bConn = conn
				log.Println("注册了B客户端")
			} else {
				log.Println("注册了消费者客户端")
			}
			mu.Unlock()
			continue
		}

		// 根据消息类型做转发处理
		switch msg.Type {
		case "offer":
			// 如果发起消息的连接非B，则转发到B
			mu.Lock()
			senderRole := roles[conn]
			currentBConn := bConn
			mu.Unlock()
			if senderRole != "B" {
				if currentBConn != nil {
					if err := currentBConn.WriteJSON(msg); err != nil {
						log.Println("向B转发offer时出错：", err)
					} else {
						log.Println("成功将offer转发给B")
					}
				} else {
					log.Println("目前没有注册的B客户端，无法转发offer")
				}
			} else {
				// 如果B发出offer，可根据需要转发给所有消费者（此处按业务需求处理）
				mu.Lock()
				for client, role := range roles {
					if client != conn && role != "B" {
						if err := client.WriteJSON(msg); err != nil {
							log.Println("向消费者转发offer时出错：", err)
						}
					}
				}
				mu.Unlock()
			}
		case "answer":
			// 如果B发出answer，则转发给所有消费者
			mu.Lock()
			for client, role := range roles {
				if client != conn && role != "B" {
					if err := client.WriteJSON(msg); err != nil {
						log.Println("向消费者转发answer时出错：", err)
					}
				}
			}
			mu.Unlock()
		case "candidate":
			// ICE候选信息，根据发消息方决定转发对象：
			mu.Lock()
			senderRole := roles[conn]
			currentBConn := bConn
			mu.Unlock()
			if senderRole != "B" {
				// 消费者发送 ICE 信息，转发给 B
				if currentBConn != nil {
					if err := currentBConn.WriteJSON(msg); err != nil {
						log.Println("向B转发candidate时出错：", err)
					}
				} else {
					log.Println("目前没有注册的B客户端，无法转发candidate")
				}
			} else {
				// 如果B发出 candidate，则转发给所有消费者
				mu.Lock()
				for client, role := range roles {
					if client != conn && role != "B" {
						if err := client.WriteJSON(msg); err != nil {
							log.Println("向消费者转发candidate时出错：", err)
						}
					}
				}
				mu.Unlock()
			}
		}
	}
}

func main() {
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			// 允许跨域
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("WebSocket升级失败：", err)
			return
		}
		handleWebSocket(conn)
	})

	fmt.Println("信令服务器启动，监听 :8090")
	if err := http.ListenAndServe(":8090", nil); err != nil {
		log.Fatal("启动服务器时出错：", err)
	}
}

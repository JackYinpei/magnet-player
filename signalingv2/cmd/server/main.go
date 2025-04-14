package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

// Message 定义消息结构，新增 Role 字段
type Message struct {
	Type      string `json:"type"`
	SDP       string `json:"sdp,omitempty"`
	Candidate string `json:"candidate,omitempty"`
	FilePath  string `json:"file_path,omitempty"` // 文件路径
	Role      string `json:"role,omitempty"`      // 用于注册时指明身份
}

// 连接到信令服务器
func connectToSignalingServer() (*websocket.Conn, error) {
	signalingServerURL := "ws://43.156.74.32:8090/ws" // 信令服务器地址
	conn, _, err := websocket.DefaultDialer.Dial(signalingServerURL, nil)
	if err != nil {
		return nil, fmt.Errorf("连接信令服务器失败: %v", err)
	}
	return conn, nil
}

// 创建 WebRTC PeerConnection
func createPeerConnection(conn *websocket.Conn) (*webrtc.PeerConnection, error) {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		fmt.Println("新建p2p失败，因为", err)
	}

	// 配置 ICE 候选回调
	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate != nil {
			candidateJSON, _ := json.Marshal(candidate.ToJSON())
			msg := Message{
				Type:      "candidate",
				Candidate: string(candidateJSON),
			}
			if err := conn.WriteJSON(msg); err != nil {
				log.Println("发送 ICE Candidate 失败:", err)
			} else {
				log.Println("ICE Candidate 发送成功")
			}
		}
	})

	// 监听 DataChannel，接收消费者发送的文件路径
	peerConnection.OnDataChannel(func(dc *webrtc.DataChannel) {
		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			var filePath string
			err := json.Unmarshal(msg.Data, &filePath)
			if err != nil {
				log.Println("反序列化文件路径失败:", err)
				return
			}
			// 根据文件路径读取文件并通过 DataChannel 发送给消费者
			sendFileToPeer(peerConnection, filePath)
		})
	})

	return peerConnection, nil
}

// 处理来自信令服务器的消息
func handleWebSocketMessages(conn *websocket.Conn, peerConnection *webrtc.PeerConnection) {
	for {
		var msg Message
		if err := conn.ReadJSON(&msg); err != nil {
			log.Println("读取 JSON 消息失败:", err)
			break
		}

		switch msg.Type {
		case "offer":
			// 将接收到的 offer 设置为远端描述
			offer := webrtc.SessionDescription{
				Type: webrtc.SDPTypeOffer,
				SDP:  msg.SDP,
			}
			if err := peerConnection.SetRemoteDescription(offer); err != nil {
				log.Fatal("设置远端描述失败:", err)
			}

			// 收到消费者发来的 offer 后，创建 answer 并返回
			answer, err := peerConnection.CreateAnswer(nil)
			if err != nil {
				log.Fatal("创建 answer 失败:", err)
			}

			if err := peerConnection.SetLocalDescription(answer); err != nil {
				log.Fatal("设置本地描述失败:", err)
			}

			// 构建 answer 消息后发送
			answerMsg := Message{
				Type: "answer",
				SDP:  answer.SDP,
			}
			if err := conn.WriteJSON(answerMsg); err != nil {
				log.Println("发送 answer 失败:", err)
			}
		case "candidate":
			// 处理 ICE 候选信息
			var iceCandidate webrtc.ICECandidateInit
			if err := json.Unmarshal([]byte(msg.Candidate), &iceCandidate); err == nil {
				peerConnection.AddICECandidate(iceCandidate)
			}
		}
	}
}

// 通过 DataChannel 发送文件
func sendFileToPeer(peerConnection *webrtc.PeerConnection, filePath string) {
	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		log.Println("打开文件失败:", err)
		return
	}
	// 注意：不要在这里关闭文件（移除 defer file.Close()）

	// 创建 DataChannel，用于文件传输
	dataChannel, err := peerConnection.CreateDataChannel("fileTransfer", nil)
	if err != nil {
		log.Println("创建 DataChannel 失败:", err)
		file.Close() // 若出错，则及时关闭文件
		return
	}

	dataChannel.OnOpen(func() {
		fmt.Println("DataChannel 已打开")
		// 直接调用 sendFile，不需用 goroutine 包裹（或可根据需求调整）
		sendFile(file, dataChannel)
	})
}

func sendFile(file *os.File, dataChannel *webrtc.DataChannel) {
	buffer := make([]byte, 1024*64) // 分块大小 64KB
	for {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Println("读取文件失败:", err)
			break
		}
		if err := dataChannel.Send(buffer[:n]); err != nil {
			log.Println("发送文件块失败:", err)
			break
		}
	}
	// 文件传输完成后，发送文件结束标志信号给 C
	endMsg := "FILE_END"
	if err := dataChannel.Send([]byte(endMsg)); err != nil {
		log.Println("发送文件结束标志失败:", err)
	} else {
		log.Println("成功发送文件结束标志")
	}
	fmt.Println("文件传输完成")
}

func main() {
	// 连接到信令服务器
	conn, err := connectToSignalingServer()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("连接信令服务器成功")

	// 发送注册消息，通知信令服务器本客户端为 B
	regMsg := Message{
		Type: "register",
		Role: "B",
	}
	if err := conn.WriteJSON(regMsg); err != nil {
		log.Fatal("注册失败:", err)
	}

	// 创建 WebRTC PeerConnection
	peerConnection, err := createPeerConnection(conn)
	if err != nil {
		log.Fatal(err)
	}

	// 后台处理信令消息
	go handleWebSocketMessages(conn, peerConnection)

	// 防止程序退出
	select {}
}

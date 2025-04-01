package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

var (
	signalServer = flag.String("server", "shiying.sh.cn:8090", "Signaling server address")
	clientID     = flag.String("id", "producer-"+fmt.Sprint(time.Now().Unix()), "Client ID")
	baseDir      = flag.String("basedir", "/root/magnet-player/backend/data", "Base directory for video files")
	chunkSize    = flag.Int("chunk", 2<<10, "Size of video chunks to send in bytes")
)

// Message represents the structure of messages exchanged with the signaling server
type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// Connection represents a WebRTC connection to a consumer
type Connection struct {
	PeerConnection *webrtc.PeerConnection
	DataChannel    *webrtc.DataChannel
	ConsumerID     string
	Active         bool
}

// ConnectionManager manages multiple WebRTC connections
type ConnectionManager struct {
	connections map[string]*Connection
	mutex       sync.Mutex
	api         *webrtc.API
	wsConn      *websocket.Conn
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(api *webrtc.API, wsConn *websocket.Conn) *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[string]*Connection),
		api:         api,
		wsConn:      wsConn,
	}
}

// CreateConnection creates a new WebRTC connection for a consumer
func (cm *ConnectionManager) CreateConnection(consumerID string) (*Connection, error) {
	// 基本ICE配置
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	// 创建PeerConnection
	peerConnection, err := cm.api.NewPeerConnection(config)
	if err != nil {
		return nil, fmt.Errorf("创建PeerConnection失败: %v", err)
	}

	// 创建数据通道
	dataChannel, err := peerConnection.CreateDataChannel("data", nil)
	if err != nil {
		peerConnection.Close()
		return nil, fmt.Errorf("创建数据通道失败: %v", err)
	}

	// 创建连接对象
	conn := &Connection{
		PeerConnection: peerConnection,
		DataChannel:    dataChannel,
		ConsumerID:     consumerID,
		Active:         true,
	}

	// 数据通道事件处理
	dataChannel.OnOpen(func() {
		log.Printf("数据通道已打开，客户端ID: %s", consumerID)
	})

	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		// 收到文件路径请求
		filePath := string(msg.Data)
		log.Printf("收到文件请求，客户端ID: %s，文件: %s", consumerID, filePath)

		// 处理视频请求
		go processVideoRequest(dataChannel, filePath)
	})

	dataChannel.OnClose(func() {
		log.Printf("数据通道已关闭，客户端ID: %s", consumerID)
		cm.mutex.Lock()
		if conn, exists := cm.connections[consumerID]; exists {
			conn.Active = false
		}
		cm.mutex.Unlock()
	})

	// ICE候选事件处理
	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}

		// 发送ICE候选到消费者
		candidateJSON := candidate.ToJSON()
		log.Printf("发送ICE候选到客户端: %s", consumerID)
		cm.sendSignalingMessage("ice-candidate", candidateJSON, consumerID)
	})

	// 连接状态监控
	peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("连接状态变更为 %s，客户端ID: %s", state.String(), consumerID)
	})

	cm.connections[consumerID] = conn
	return conn, nil
}

// ProcessSignalingMessage 处理信令消息
func (cm *ConnectionManager) ProcessSignalingMessage(msg Message, senderID string) {
	switch msg.Type {
	case "connect":
		log.Printf("收到连接请求，客户端ID: %s", senderID)

		// 检查是否已存在此消费者的连接，避免重复处理
		cm.mutex.Lock()
		conn, exists := cm.connections[senderID]
		activeExists := exists && conn.Active
		cm.mutex.Unlock()

		if activeExists {
			log.Printf("已存在与客户端 %s 的活跃连接，忽略重复的连接请求", senderID)
			return
		}

		// 创建新连接
		conn, err := cm.CreateConnection(senderID)
		if err != nil {
			log.Printf("创建连接失败: %v", err)
			return
		}

		// 创建SDP offer
		offer, err := conn.PeerConnection.CreateOffer(nil)
		if err != nil {
			log.Printf("创建offer失败: %v", err)
			return
		}

		// 设置本地描述
		err = conn.PeerConnection.SetLocalDescription(offer)
		if err != nil {
			log.Printf("设置本地描述失败: %v", err)
			return
		}

		// 发送offer给消费者
		offerData := map[string]interface{}{
			"sdp":      offer.SDP,
			"type":     offer.Type.String(),
			"clientId": senderID, // 使用clientId，确保与前端代码一致
		}

		log.Printf("发送offer给客户端: %s", senderID)
		cm.sendSignalingMessage("offer", offerData, senderID)

	case "answer":
		log.Printf("收到answer，客户端ID: %s", senderID)

		// 解析SDP answer
		var sdpObj map[string]interface{}
		var sdp string

		// 处理不同格式的answer数据
		if dataStr, ok := msg.Data.(string); ok {
			if err := json.Unmarshal([]byte(dataStr), &sdpObj); err != nil {
				log.Printf("解析SDP answer字符串失败: %v", err)
				return
			}
			if sdpValue, ok := sdpObj["sdp"].(string); ok {
				sdp = sdpValue
			}
		} else if dataMap, ok := msg.Data.(map[string]interface{}); ok {
			if sdpValue, ok := dataMap["sdp"].(string); ok {
				sdp = sdpValue
			}
			sdpObj = dataMap
		}

		if sdp == "" {
			log.Printf("无法从answer中获取SDP")
			return
		}

		// 查找对应的连接
		cm.mutex.Lock()
		conn, exists := cm.connections[senderID]
		cm.mutex.Unlock()

		if !exists || !conn.Active {
			log.Printf("找不到活跃的连接: %s", senderID)
			return
		}

		// 设置远程描述
		err := conn.PeerConnection.SetRemoteDescription(webrtc.SessionDescription{
			Type: webrtc.SDPTypeAnswer,
			SDP:  sdp,
		})

		if err != nil {
			log.Printf("设置远程描述失败: %v", err)
			return
		}

		log.Printf("设置远程描述成功，客户端ID: %s", senderID)

	case "ice-candidate":
		log.Printf("收到ICE候选，客户端ID: %s", senderID)

		// 查找对应的连接
		cm.mutex.Lock()
		conn, exists := cm.connections[senderID]
		cm.mutex.Unlock()

		if !exists || !conn.Active {
			log.Printf("找不到活跃的连接: %s", senderID)
			return
		}

		// 解析ICE候选
		var candidate map[string]interface{}

		if dataStr, ok := msg.Data.(string); ok {
			if err := json.Unmarshal([]byte(dataStr), &candidate); err != nil {
				log.Printf("解析ICE候选字符串失败: %v", err)
				return
			}
		} else if dataMap, ok := msg.Data.(map[string]interface{}); ok {
			candidate = dataMap
		} else {
			log.Printf("未知的ICE候选数据格式")
			return
		}

		candidateStr, ok := candidate["candidate"].(string)
		if !ok {
			log.Printf("ICE候选数据格式错误")
			return
		}

		// 添加ICE候选
		err := conn.PeerConnection.AddICECandidate(webrtc.ICECandidateInit{
			Candidate: candidateStr,
		})

		if err != nil {
			log.Printf("添加ICE候选失败: %v", err)
			return
		}

	default:
		log.Printf("收到未知类型的消息: %s", msg.Type)
	}
}

// Send a message to the signaling server
func (cm *ConnectionManager) sendSignalingMessage(msgType string, data interface{}, targetID string) {
	msg := Message{
		Type: msgType,
		Data: data,
	}

	// 不做额外的ID处理，避免重复
	// 数据中应该已经包含targetId

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error encoding message: %v", err)
		return
	}

	// 发送到信令服务器
	if err := cm.wsConn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
		log.Printf("Error sending message to signaling server: %v", err)
	}
}

// CloseAllConnections closes all WebRTC connections
func (cm *ConnectionManager) CloseAllConnections() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	for _, conn := range cm.connections {
		if conn.Active {
			conn.PeerConnection.Close()
			conn.Active = false
		}
	}
}

func main() {
	flag.Parse()

	// Create a new WebRTC API with default codecs
	api := webrtc.NewAPI()

	// Connect to the signaling server
	u := url.URL{
		Scheme:   "wss",
		Host:     *signalServer,
		Path:     "/ws",
		RawQuery: fmt.Sprintf("id=%s&type=producer", *clientID),
	}
	log.Printf("Connecting to signaling server: %s", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatalf("Failed to connect to signaling server: %v", err)
	}
	defer conn.Close()
	log.Println("Connected to signaling server")

	// Create connection manager
	connectionManager := NewConnectionManager(api, conn)
	defer connectionManager.CloseAllConnections()

	// Handle incoming signaling messages
	go func() {
		for {
			_, msgBytes, err := conn.ReadMessage()
			if err != nil {
				log.Printf("Error reading from signaling server: %v", err)
				return
			}

			// 由于信令服务器发送的是原始消息，不是封装结构
			var msg Message
			if err := json.Unmarshal(msgBytes, &msg); err != nil {
				log.Printf("Error parsing message: %v", err)
				continue
			}

			// 从消息中提取发送者ID
			var senderID string

			// 尝试从消息数据中提取clientId
			var data map[string]interface{}
			if dataStr, ok := msg.Data.(string); ok {
				// 如果Data是字符串，尝试解析成map
				if err := json.Unmarshal([]byte(dataStr), &data); err == nil {
					if clientID, ok := data["clientId"].(string); ok && clientID != "" {
						senderID = clientID
						log.Printf("Extracted client ID from message: %s", senderID)
					}
				}
			} else if dataMap, ok := msg.Data.(map[string]interface{}); ok {
				// 如果Data已经是map，直接尝试获取clientId
				if clientID, ok := dataMap["clientId"].(string); ok && clientID != "" {
					senderID = clientID
					log.Printf("Extracted client ID from message data map: %s", senderID)
				}
			}

			// 如果无法从消息中提取ID，则使用之前的逻辑
			if senderID == "" {
				if msg.Type == "offer" {
					// 为新的offer生成一个临时ID
					senderID = fmt.Sprintf("consumer-%d", time.Now().Unix())
					log.Printf("No ID in message, assigning temporary ID: %s", senderID)
				} else {
					// 对于其他消息类型，尝试根据活跃连接匹配
					connectionManager.mutex.Lock()
					for id, conn := range connectionManager.connections {
						if conn.Active {
							senderID = id
							break
						}
					}
					connectionManager.mutex.Unlock()

					if senderID == "" {
						// 如果找不到活跃连接，使用一个默认值
						senderID = "unknown-consumer"
						log.Printf("Could not determine sender ID, using default: %s", senderID)
					}
				}
			}

			// 处理消息
			connectionManager.ProcessSignalingMessage(msg, senderID)
		}
	}()

	// Start a goroutine to read from stdin for commands
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Println("Producer client started. Enter 'list' to see active connections or 'exit' to quit:")
		for scanner.Scan() {
			cmd := scanner.Text()
			switch cmd {
			case "list":
				connectionManager.mutex.Lock()
				fmt.Printf("Active connections: %d\n", len(connectionManager.connections))
				for id, conn := range connectionManager.connections {
					if conn.Active {
						fmt.Printf("- Consumer %s: active\n", id)
					} else {
						fmt.Printf("- Consumer %s: inactive\n", id)
					}
				}
				connectionManager.mutex.Unlock()
			case "exit":
				os.Exit(0)
			default:
				fmt.Println("Unknown command. Available commands: 'list', 'exit'")
			}
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt

	// Close the peer connection
	log.Println("Shutting down...")
}

func processVideoRequest(dataChannel *webrtc.DataChannel, requestedPath string) {
	// Sanitize the requested path to prevent directory traversal
	cleanPath := filepath.Clean(requestedPath)

	// Prevent directory traversal by ensuring the path doesn't contain ".."
	if filepath.IsAbs(cleanPath) || cleanPath == ".." || filepath.HasPrefix(cleanPath, ".."+string(filepath.Separator)) {
		sendErrorMessage(dataChannel, "Invalid path: directory traversal attempt detected")
		return
	}

	// Construct the absolute file path
	filePath := filepath.Join(*baseDir, cleanPath)

	// Check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		sendErrorMessage(dataChannel, fmt.Sprintf("File not found: %s", cleanPath))
		return
	}

	// Send the video file
	log.Printf("Sending video file: %s", filePath)
	if err := sendVideoFile(dataChannel, filePath); err != nil {
		log.Printf("Error sending video file: %v", err)
		sendErrorMessage(dataChannel, fmt.Sprintf("Error sending video: %v", err))
	}
}

func sendVideoFile(dataChannel *webrtc.DataChannel, filePath string) error {
	// Open the video file
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}
	fileSize := fileInfo.Size()

	// Send file metadata
	metadata := struct {
		Type     string `json:"type"`
		FileName string `json:"fileName"`
		FileSize int64  `json:"fileSize"`
	}{
		Type:     "metadata",
		FileName: filepath.Base(filePath),
		FileSize: fileSize,
	}

	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	if err := dataChannel.Send(metadataBytes); err != nil {
		return err
	}
	log.Printf("Sent file metadata: %s, size: %d bytes", filepath.Base(filePath), fileSize)

	// Read and send the file in chunks
	buffer := make([]byte, *chunkSize)
	totalSent := 0
	startTime := time.Now()

	for {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Create chunk message
		chunkMsg := struct {
			Type      string `json:"type"`
			ChunkData []byte `json:"chunkData"`
		}{
			Type:      "chunk",
			ChunkData: buffer[:n],
		}

		chunkBytes, err := json.Marshal(chunkMsg)
		if err != nil {
			return err
		}

		// Send the chunk
		if err := dataChannel.Send(chunkBytes); err != nil {
			return err
		}

		totalSent += n
		elapsed := time.Since(startTime).Seconds()
		if elapsed > 0 {
			speed := float64(totalSent) / elapsed / 1024 / 1024
			log.Printf("Sent %d/%d bytes (%.2f%%) at %.2f MB/s",
				totalSent, fileSize, float64(totalSent)*100/float64(fileSize), speed)
		}

		// Add a small delay to prevent overwhelming the channel
		time.Sleep(5 * time.Millisecond)
	}

	// Send end-of-file message
	eofMsg := struct {
		Type string `json:"type"`
	}{
		Type: "eof",
	}

	eofBytes, err := json.Marshal(eofMsg)
	if err != nil {
		return err
	}

	if err := dataChannel.Send(eofBytes); err != nil {
		return err
	}
	log.Printf("File transfer complete: %s", filepath.Base(filePath))

	return nil
}

func sendErrorMessage(dataChannel *webrtc.DataChannel, errMsg string) {
	errMsgStruct := struct {
		Type  string `json:"type"`
		Error string `json:"error"`
	}{
		Type:  "error",
		Error: errMsg,
	}

	msgBytes, _ := json.Marshal(errMsgStruct)
	if err := dataChannel.Send(msgBytes); err != nil {
		log.Printf("Error sending error message: %v", err)
	}
}

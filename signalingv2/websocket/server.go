package main

import (
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	// 打开本地 H264 文件，逐帧发送
	filename := flag.Lookup("file").Value.String()
	file, err := os.Open(filename)
	if err != nil {
		log.Println("Open file error:", err)
		return
	}
	defer file.Close()

	buf := make([]byte, 2<<20) // 一次发送 2MB
	for {
		n, err := file.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Println("Read error:", err)
			return
		}
		err = conn.WriteMessage(websocket.BinaryMessage, buf[:n])
		if err != nil {
			log.Println("WebSocket write error:", err)
			return
		}
		time.Sleep(33 * time.Millisecond) // 约30fps
	}
}

func main() {
	flag.String("file", "test.h264", "H264 file to send")
	flag.Parse()

	http.HandleFunc("/ws", wsHandler)
	log.Println("WebSocket server started at :8080/ws")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

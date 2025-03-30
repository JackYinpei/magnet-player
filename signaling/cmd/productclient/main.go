package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

var (
	signalServer = flag.String("server", "43.156.74.32:8090", "Signaling server address")
	clientID     = flag.String("id", "producer-"+fmt.Sprint(time.Now().Unix()), "Client ID")
)

// Message represents the structure of messages exchanged with the signaling server
type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

func main() {
	flag.Parse()

	// Create a new WebRTC API with default codecs
	api := webrtc.NewAPI()

	// Create a new RTCPeerConnection
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	peerConnection, err := api.NewPeerConnection(config)
	if err != nil {
		log.Fatalf("Failed to create peer connection: %v", err)
	}
	defer peerConnection.Close()

	// Create a data channel
	dataChannel, err := peerConnection.CreateDataChannel("data", nil)
	if err != nil {
		log.Fatalf("Failed to create data channel: %v", err)
	}

	// Set up data channel event handlers
	dataChannel.OnOpen(func() {
		log.Println("Data channel opened")
		
		// After the channel is open, we can send messages
		// Start a goroutine to read from stdin and send messages
		go func() {
			scanner := bufio.NewScanner(os.Stdin)
			fmt.Println("Data channel connected. Enter messages to send to consumer:")
			for scanner.Scan() {
				msg := scanner.Text()
				if err := dataChannel.SendText(msg); err != nil {
					log.Printf("Failed to send message: %v", err)
				} else {
					log.Printf("Sent message: %s", msg)
				}
			}
		}()
	})

	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		log.Printf("Received message from consumer: %s", string(msg.Data))
	})

	dataChannel.OnClose(func() {
		log.Println("Data channel closed")
	})

	// Connect to the signaling server
	u := url.URL{
		Scheme:   "ws",
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

	// Global websocket connection for signaling
	var wsConn = conn

	// Helper function to send messages to the signaling server
	sendSignalingMessage := func(msgType string, data interface{}) {
		msg := Message{
			Type: msgType,
			Data: data,
		}

		msgBytes, err := json.Marshal(msg)
		if err != nil {
			log.Printf("Error encoding message: %v", err)
			return
		}

		if err := wsConn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
			log.Printf("Error sending message to signaling server: %v", err)
		}
	}

	// ICE candidate handler
	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}

		// Send ICE candidate to signaling server
		sendSignalingMessage("ice-candidate", candidate.ToJSON())
	})

	// Handle incoming signaling messages
	go func() {
		for {
			_, msgBytes, err := conn.ReadMessage()
			if err != nil {
				log.Printf("Error reading from signaling server: %v", err)
				return
			}

			var msg Message
			if err := json.Unmarshal(msgBytes, &msg); err != nil {
				log.Printf("Error parsing message: %v", err)
				continue
			}

			switch msg.Type {
			case "offer":
				log.Println("Received offer (unexpected for producer)")
				
			case "answer":
				// Handle answer from consumer
				var sdp webrtc.SessionDescription
				data, _ := json.Marshal(msg.Data)
				if err := json.Unmarshal(data, &sdp); err != nil {
					log.Printf("Error parsing SDP answer: %v", err)
					continue
				}

				if err := peerConnection.SetRemoteDescription(sdp); err != nil {
					log.Printf("Error setting remote description: %v", err)
				}
				
			case "ice-candidate":
				// Handle ICE candidate from consumer
				var candidate webrtc.ICECandidateInit
				data, _ := json.Marshal(msg.Data)
				if err := json.Unmarshal(data, &candidate); err != nil {
					log.Printf("Error parsing ICE candidate: %v", err)
					continue
				}

				if err := peerConnection.AddICECandidate(candidate); err != nil {
					log.Printf("Error adding ICE candidate: %v", err)
				}
			}
		}
	}()

	// Create an offer
	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		log.Fatalf("Failed to create offer: %v", err)
	}

	// Set local description
	if err := peerConnection.SetLocalDescription(offer); err != nil {
		log.Fatalf("Failed to set local description: %v", err)
	}

	// Send offer to signaling server
	sendSignalingMessage("offer", offer)

	// Wait for interrupt signal to gracefully shutdown
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt
	log.Println("Shutting down...")
}

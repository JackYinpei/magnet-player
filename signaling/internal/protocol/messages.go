package protocol

// MessageType defines the type of message being sent
type MessageType string

const (
	// Register message is sent by clients and servers to register with the signaling server
	Register MessageType = "register"
	// ListServers message is sent by clients to request available servers
	ListServers MessageType = "list-servers"
	// ServerList message is sent by the signaling server with a list of available servers
	ServerList MessageType = "server-list"
	// ConnectRequest message is sent by clients to request connection to a server
	ConnectRequest MessageType = "connect-request"
	// ConnectResponse message is sent by the signaling server to initiate connection between client and server
	ConnectResponse MessageType = "connect-response"
	// ICECandidate message is sent for WebRTC ICE candidate exchange
	ICECandidate MessageType = "ice-candidate"
	// SDPOffer message contains a WebRTC SDP offer
	SDPOffer MessageType = "sdp-offer"
	// SDPAnswer message contains a WebRTC SDP answer
	SDPAnswer MessageType = "sdp-answer"
	// DataRequest message is sent by clients to request data from a server
	DataRequest MessageType = "data-request"
	// DataResponse message is sent by servers with the requested data
	DataResponse MessageType = "data-response"
)

// Message is the basic message structure for all communication
type Message struct {
	Type    MessageType `json:"type"`
	Payload interface{} `json:"payload"`
}

// RegisterMessage is sent when a client or server registers with the signaling server
type RegisterMessage struct {
	ID   string `json:"id"`
	Role string `json:"role"` // "server" or "client"
}

// ServerInfo contains information about a registered server
type ServerInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ServerListMessage contains a list of available servers
type ServerListMessage struct {
	Servers []ServerInfo `json:"servers"`
}

// ConnectRequestMessage is sent by a client to request connection to a specific server
type ConnectRequestMessage struct {
	ServerID string `json:"server_id"`
	ClientID string `json:"client_id"`
}

// ConnectResponseMessage is sent to both client and server to initiate direct connection
type ConnectResponseMessage struct {
	ServerID string `json:"server_id"`
	ClientID string `json:"client_id"`
	Success  bool   `json:"success"`
	Error    string `json:"error,omitempty"`
}

// ICECandidateMessage contains WebRTC ICE candidates
type ICECandidateMessage struct {
	Target    string      `json:"target"` // ID of the target peer
	Candidate interface{} `json:"candidate"`
}

// SDPMessage contains WebRTC session description
type SDPMessage struct {
	Target string `json:"target"` // ID of the target peer
	SDP    string `json:"sdp"`
}

// DataRequestMessage is sent by clients to request specific data
type DataRequestMessage struct {
	Path string `json:"path"`
}

// DataResponseMessage is sent by servers with the requested data
type DataResponseMessage struct {
	Path    string `json:"path"`
	Data    []byte `json:"data"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

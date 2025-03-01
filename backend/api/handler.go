package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/torrentplayer/backend/torrent"
)

// Handler handles API requests
type Handler struct {
	torrentClient *torrent.Client
}

// NewHandler creates a new API handler
func NewHandler(torrentClient *torrent.Client) *Handler {
	return &Handler{
		torrentClient: torrentClient,
	}
}

// AddMagnet handles requests to add a magnet link
func (h *Handler) AddMagnet(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the request body
	var req struct {
		MagnetURI string `json:"magnetUri"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Add the magnet link
	info, err := h.torrentClient.AddMagnet(req.MagnetURI)
	if err != nil {
		http.Error(w, "Failed to add magnet link: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the torrent info
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// ListTorrents handles requests to list all torrents
func (h *Handler) ListTorrents(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get the list of torrents
	torrents := h.torrentClient.ListTorrents()

	// Return the torrents
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(torrents)
}

// ListFiles handles requests to list all files in a torrent
func (h *Handler) ListFiles(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get the info hash from the URL query
	infoHash := r.URL.Query().Get("infoHash")
	if infoHash == "" {
		http.Error(w, "Missing infoHash parameter", http.StatusBadRequest)
		return
	}

	// Get the list of files
	files, err := h.torrentClient.ListFiles(infoHash)
	if err != nil {
		http.Error(w, "Failed to list files: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the files
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

// StreamFile handles requests to stream a file from a torrent
func (h *Handler) StreamFile(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Range, Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract torrent info hash and file index from the URL path
	// Expected format: /stream/{infoHash}/{fileIndex}
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid path format", http.StatusBadRequest)
		return
	}

	infoHash := pathParts[2]
	fileIndexStr := pathParts[3]

	// Parse the file index
	fileIndex, err := strconv.Atoi(fileIndexStr)
	if err != nil {
		http.Error(w, "Invalid file index", http.StatusBadRequest)
		return
	}

	// Get the torrent
	t, ok := h.torrentClient.GetTorrent(infoHash)
	if !ok {
		http.Error(w, "Torrent not found", http.StatusNotFound)
		return
	}

	// Check if the file index is valid
	if fileIndex < 0 || fileIndex >= len(t.Files()) {
		http.Error(w, "File index out of range", http.StatusBadRequest)
		return
	}

	// Get the file
	file := t.Files()[fileIndex]

	// Set content type based on file extension
	contentType := getContentTypeFromPath(file.DisplayPath())
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Length", strconv.FormatInt(file.Length(), 10))

	// Handle range requests
	var start, end int64
	rangeHeader := r.Header.Get("Range")
	if rangeHeader != "" {
		rangeParts := strings.Split(strings.TrimPrefix(rangeHeader, "bytes="), "-")
		if len(rangeParts) == 2 {
			start, _ = strconv.ParseInt(rangeParts[0], 10, 64)
			if rangeParts[1] != "" {
				end, _ = strconv.ParseInt(rangeParts[1], 10, 64)
			} else {
				end = file.Length() - 1
			}
		}

		// Ensure end is valid
		if end >= file.Length() {
			end = file.Length() - 1
		}

		// Set partial content headers
		w.Header().Set("Content-Range", "bytes "+strconv.FormatInt(start, 10)+"-"+strconv.FormatInt(end, 10)+"/"+strconv.FormatInt(file.Length(), 10))
		w.Header().Set("Content-Length", strconv.FormatInt(end-start+1, 10))
		w.WriteHeader(http.StatusPartialContent)
	} else {
		start = 0
		end = file.Length() - 1
	}

	// Create a reader for the file
	reader := file.NewReader()
	defer reader.Close()

	// Seek to the start position
	_, err = reader.Seek(start, io.SeekStart)
	if err != nil {
		http.Error(w, "Failed to seek: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Stream the file
	_, err = io.CopyN(w, reader, end-start+1)
	if err != nil && err != io.EOF {
		// Just log the error, the client might have disconnected
		return
	}
}

// getContentTypeFromPath determines the content type of a file based on its path
func getContentTypeFromPath(path string) string {
	ext := strings.ToLower(path[strings.LastIndex(path, ".")+1:])
	switch ext {
	case "mp4":
		return "video/mp4"
	case "webm":
		return "video/webm"
	case "mkv":
		return "video/x-matroska"
	case "avi":
		return "video/x-msvideo"
	case "mov":
		return "video/quicktime"
	case "wmv":
		return "video/x-ms-wmv"
	case "flv":
		return "video/x-flv"
	case "m4v":
		return "video/x-m4v"
	case "mpg", "mpeg":
		return "video/mpeg"
	case "3gp":
		return "video/3gpp"
	default:
		return "application/octet-stream"
	}
}

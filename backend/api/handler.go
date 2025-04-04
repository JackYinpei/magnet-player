package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/torrentplayer/backend/db"
	"github.com/torrentplayer/backend/torrent"
)

// Handler handles API requests
type Handler struct {
	torrentClient *torrent.Client
	torrentStore  *db.TorrentStore
}

// NewHandler creates a new API handler
func NewHandler(torrentClient *torrent.Client, torrentStore *db.TorrentStore) *Handler {
	return &Handler{
		torrentClient: torrentClient,
		torrentStore:  torrentStore,
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

	// Save the torrent to the database
	record := db.TorrentRecord{
		InfoHash:  info.InfoHash,
		Name:      info.Name,
		MagnetURI: req.MagnetURI,
		AddedAt:   info.AddedAt,
	}

	if err := h.torrentStore.AddTorrent(&record); err != nil {
		log.Printf("Failed to save torrent to database: %v", err)
	}

	// Return the torrent info
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// ListTorrents handles requests to list all torrents, just torrent client status
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

	// Get the list of torrents from the client
	torrents := h.torrentClient.ListTorrents()
	// Return the torrents
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(torrents)
}

// UpdateMovieDetails handles requests to update movie details for a torrent
func (h *Handler) UpdateMovieDetails(w http.ResponseWriter, r *http.Request) {
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

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		http.Error(w, "Missing info hash", http.StatusBadRequest)
		return
	}
	infoHash := parts[len(parts)-1]

	// Parse the request body
	var movieDetails db.MovieDetails
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if err := json.Unmarshal(body, &movieDetails); err != nil {
		http.Error(w, "Failed to parse request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Retrieve the torrent record from the database
	record, err := h.torrentStore.GetTorrent(infoHash)
	if err != nil {
		log.Printf("Torrent not found in database: %v", err)

		// 创建一个新的记录 - 只包含电影详情，不包含磁力链接
		// 这样可以避免启动时恢复种子时尝试使用无效的磁力链接
		// currentTime := time.Now()
		// record = &db.TorrentRecord{
		// 	InfoHash:     infoHash,
		// 	Name:         movieDetails.OriginalTitle,
		// 	MagnetURI:    "", // 空的磁力链接，但不会尝试添加它
		// 	AddedAt:      currentTime,
		// 	MovieDetails: &movieDetails,
		// }

		// 保存新记录到数据库
		// if err := h.torrentStore.AddTorrent(record); err != nil {
		// 	http.Error(w, "Failed to save torrent record: "+err.Error(), http.StatusInternalServerError)
		// 	return
		// }
	} else {
		// 更新电影详情，保留原有的磁力链接和其他信息
		record.MovieDetails = &movieDetails

		// 保存更新后的记录到数据库
		if err := h.torrentStore.UpdateTorrentMovieDetail(record); err != nil {
			http.Error(w, "Failed to save movie details: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
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

	fmt.Println(r.URL.Path, "StreamFile file path")

	// Extract torrent info hash and file name from the URL path
	// Expected format: /stream/{infoHash}/{fileName}
	pathParts := strings.Split(r.URL.Path, "/")
	infoHash := pathParts[3]
	fileName := pathParts[4]

	fmt.Println(infoHash, fileName, "StreamFile infoHash and fileName", pathParts)

	// Get the torrent
	t, ok := h.torrentClient.GetTorrent(infoHash)
	if !ok {
		fmt.Println("Torrent not found", infoHash)
		http.Error(w, "Torrent not found", http.StatusNotFound)
		return
	}

	found := false
	var fileIndex int

	// 获取文件列表
	filesList, listErr := h.torrentClient.ListFiles(infoHash)
	if listErr != nil {
		fmt.Println("Failed to list files", listErr)
		http.Error(w, "Failed to list files: "+listErr.Error(), http.StatusInternalServerError)
		return
	}

	// 通过文件名查找匹配的文件
	for _, f := range filesList {
		if f.Path == fileName {
			fmt.Println("File found", f)
			fileIndex = f.FileIndex
			found = true
			break
		}
	}

	if !found {
		fmt.Println("File not found", fileName)
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// 获取原始文件对象
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
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, file.Length()))
		w.Header().Set("Content-Length", strconv.FormatInt(end-start+1, 10))
		w.WriteHeader(http.StatusPartialContent)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	// Create reader
	reader := file.NewReader()
	if reader == nil {
		http.Error(w, "Failed to create reader", http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	// Seek to start position if needed
	if start > 0 {
		if _, err := reader.Seek(start, io.SeekStart); err != nil {
			http.Error(w, "Failed to seek to position: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Stream the file
	var streamErr error
	if end > 0 {
		// Stream a range
		_, streamErr = io.CopyN(w, reader, end-start+1)
	} else {
		// Stream the whole file
		_, streamErr = io.Copy(w, reader)
	}

	if streamErr != nil {
		// Don't return an error, as the client may have disconnected
		log.Printf("Error streaming file: %v", streamErr)
	}
}

// getContentTypeFromPath determines the content type of a file based on its path
func getContentTypeFromPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".mp4", ".m4v", ".mov":
		return "video/mp4"
	case ".mkv":
		return "video/x-matroska"
	case ".avi":
		return "video/x-msvideo"
	case ".wmv":
		return "video/x-ms-wmv"
	case ".webm":
		return "video/webm"
	case ".flv":
		return "video/x-flv"
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".flac":
		return "audio/flac"
	case ".ogg":
		return "audio/ogg"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".srt":
		return "application/x-subrip"
	case ".vtt":
		return "text/vtt"
	case ".txt":
		return "text/plain"
	case ".pdf":
		return "application/pdf"
	case ".zip":
		return "application/zip"
	case ".rar":
		return "application/x-rar-compressed"
	default:
		return "application/octet-stream"
	}
}

// GetMovieDetails handles requests to get movie details for all torrents
func (h *Handler) GetMovieDetails(w http.ResponseWriter, r *http.Request) {
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

	// Get all torrents from the database with their movie details
	records, err := h.torrentStore.GetAllTorrents()
	if err != nil {
		http.Error(w, "Failed to get movie details: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the movie details
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}

// SaveTorrentData handles requests to save torrent data including file paths to the database
func (h *Handler) SaveTorrentData(w http.ResponseWriter, r *http.Request) {
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

	// Extract the infoHash from the URL path
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		http.Error(w, "Invalid URL path", http.StatusBadRequest)
		return
	}
	infoHash := parts[len(parts)-1]

	// Parse the request body
	var torrentData struct {
		InfoHash   string        `json:"infoHash"`
		Name       string        `json:"name"`
		Length     int64         `json:"length"`
		Files      []db.FileInfo `json:"files"`
		Downloaded int64         `json:"downloaded"`
		Progress   float32       `json:"progress"`
		State      string        `json:"state"`
		AddedAt    time.Time     `json:"addedAt"`
	}

	if err := json.NewDecoder(r.Body).Decode(&torrentData); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Verify infoHash matches the one in the URL
	if infoHash != torrentData.InfoHash {
		http.Error(w, "InfoHash mismatch", http.StatusBadRequest)
		return
	}

	// Extract file paths
	filePaths := make([]string, len(torrentData.Files))
	for i, file := range torrentData.Files {
		filePaths[i] = file.Path
	}

	// Serialize file paths to JSON
	dataPathJSON, err := json.Marshal(filePaths)
	if err != nil {
		http.Error(w, "Failed to serialize file paths: "+err.Error(), http.StatusInternalServerError)
		return
	}

	torrentRecord := db.TorrentRecord{
		InfoHash:   infoHash,
		Name:       torrentData.Name,
		Length:     torrentData.Length,
		Files:      torrentData.Files,
		Downloaded: torrentData.Downloaded,
		Progress:   torrentData.Progress,
		State:      torrentData.State,
		MagnetURI:  torrentData.InfoHash,
		AddedAt:    torrentData.AddedAt,
		DataPath:   string(dataPathJSON),
	}

	// Update the data_path in the database
	if err := h.torrentStore.UpdateTorrent(&torrentRecord); err != nil {
		http.Error(w, "Failed to update data path: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Torrent data saved successfully",
	})
}

package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
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
	
	if err := h.torrentStore.SaveTorrent(record); err != nil {
		log.Printf("Failed to save torrent to database: %v", err)
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

	// Get the list of torrents from the client
	torrents := h.torrentClient.ListTorrents()

	// For each torrent, check if we have saved movie details in the database
	for i, torrent := range torrents {
		record, err := h.torrentStore.GetTorrent(torrent.InfoHash)
		if err != nil {
			// If not found in DB, continue with the client's torrent info
			continue
		}

		// If we have movie details saved, merge them with the torrent info
		if record.MovieDetails != nil {
			// Update the client-side torrent with our stored movie details
			// We keep all the runtime info (progress, download status, etc.)
			// but add movie details and files info from the database
			movieDetails := record.MovieDetails
			
			// The client-side torrent already has files info in Files field
			// so we don't need to overwrite it, we just keep the movie details
			
			// Add the movie details as a custom JSON field in the response
			torrents[i].MovieDetails = movieDetails
		}
	}

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

	// 从URL路径中提取infoHash
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 6 {
		http.Error(w, "Invalid URL path", http.StatusBadRequest)
		return
	}
	infoHash := pathParts[len(pathParts)-1]
	
	if infoHash == "" {
		http.Error(w, "Missing infoHash", http.StatusBadRequest)
		return
	}

	// 解析请求体中的电影详情
	var movieDetails db.MovieDetails
	if err := json.NewDecoder(r.Body).Decode(&movieDetails); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 检查种子是否存在
	t, ok := h.torrentClient.GetTorrent(infoHash)
	if !ok {
		http.Error(w, "Torrent not found", http.StatusNotFound)
		return
	}

	// 从种子客户端获取文件信息
	clientFiles, err := h.torrentClient.ListFiles(infoHash)
	if err != nil {
		log.Printf("Failed to get files for torrent %s: %v", infoHash, err)
	} else {
		// 转换 torrent.FileInfo 到 db.FileInfo
		var dbFiles []db.FileInfo
		for _, f := range clientFiles {
			dbFiles = append(dbFiles, db.FileInfo{
				Path:       f.Path,
				Length:     f.Length,
				Progress:   f.Progress,
				FileIndex:  f.FileIndex,
				TorrentID:  f.TorrentID,
				IsVideo:    f.IsVideo,
				IsPlayable: f.IsPlayable,
			})
		}
		
		// 添加文件信息到电影详情
		movieDetails.Files = dbFiles
	}

	// 获取种子记录
	record, err := h.torrentStore.GetTorrent(infoHash)
	if err != nil {
		log.Printf("Torrent not found in database, creating new record: %v", err)
		record = db.TorrentRecord{
			InfoHash:  infoHash,
			Name:      t.Name(), // t.Name 是函数，需要调用它
			MagnetURI: "", // 客户端对象中没有此字段，设置为空
			AddedAt:   time.Now(), // 客户端对象中没有此字段，设置为当前时间
		}
	}

	// 更新电影详情
	record.MovieDetails = &movieDetails

	// 保存到数据库
	if err := h.torrentStore.SaveTorrent(record); err != nil {
		http.Error(w, "Failed to save movie details: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 返回成功响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Movie details saved successfully",
	})
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
	if end > 0 {
		// Stream a range
		_, err = io.CopyN(w, reader, end-start+1)
	} else {
		// Stream the whole file
		_, err = io.Copy(w, reader)
	}

	if err != nil {
		// Don't return an error, as the client may have disconnected
		log.Printf("Error streaming file: %v", err)
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

	// Extract only the necessary movie information for the UI
	type MovieInfo struct {
		InfoHash     string          `json:"infoHash"`
		Name         string          `json:"name"`
		AddedAt      time.Time       `json:"addedAt"`
		MovieDetails *db.MovieDetails `json:"movieDetails,omitempty"`
	}

	movieInfoList := make([]MovieInfo, 0, len(records))
	for _, record := range records {
		movieInfo := MovieInfo{
			InfoHash:     record.InfoHash,
			Name:         record.Name,
			AddedAt:      record.AddedAt,
			MovieDetails: record.MovieDetails,
		}
		movieInfoList = append(movieInfoList, movieInfo)
	}

	// Return the movie details
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(movieInfoList)
}

// SearchMovie handles requests to search for a movie by name
func (h *Handler) SearchMovie(w http.ResponseWriter, r *http.Request) {
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

	// Get the filename from the URL query
	filename := r.URL.Query().Get("filename")
	if filename == "" {
		http.Error(w, "Missing filename parameter", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual movie info lookup from an external API
	// For now, we'll return a mock response with basic movie details based on the filename
	
	// Extract the movie name and year from the filename using basic parsing
	// This is a simple implementation and might not work for all filenames
	name := filename
	year := ""
	
	// Try to extract year in format (YYYY) or .YYYY.
	yearPattern1 := strings.LastIndex(name, "(")
	yearPattern2 := strings.LastIndex(name, ".")
	
	if yearPattern1 != -1 && yearPattern1+5 <= len(name) && name[yearPattern1+1] >= '1' && name[yearPattern1+1] <= '2' {
		// Extract year from (YYYY) format
		yearStr := name[yearPattern1+1 : yearPattern1+5]
		if _, err := strconv.Atoi(yearStr); err == nil {
			year = yearStr
			name = strings.TrimSpace(name[:yearPattern1])
		}
	} else if yearPattern2 != -1 && yearPattern2+5 <= len(name) && name[yearPattern2+1] >= '1' && name[yearPattern2+1] <= '2' {
		// Extract year from .YYYY. format
		yearStr := name[yearPattern2+1 : yearPattern2+5]
		if _, err := strconv.Atoi(yearStr); err == nil {
			year = yearStr
			name = strings.TrimSpace(name[:yearPattern2])
		}
	}
	
	// Clean up the name by removing common suffixes and file extensions
	name = strings.TrimSuffix(name, ".mp4")
	name = strings.TrimSuffix(name, ".mkv")
	name = strings.TrimSuffix(name, ".avi")
	
	// Create a mock movie info response
	movieInfo := map[string]interface{}{
		"filename":     name,
		"year":         year,
		"posterUrl":    "https://via.placeholder.com/300x450?text=" + url.QueryEscape(name),
		"backdropUrl":  "https://via.placeholder.com/1280x720?text=" + url.QueryEscape(name),
		"overview":     "这是关于 " + name + " 的电影简介。",
		"rating":       5.0,
		"voteCount":    10,
		"genres":       []string{"未知"},
		"runtime":      90,
		"tmdbId":       0,
		"releaseDate":  time.Now().Format("2006-01-02"),
		"originalTitle": name,
		"popularity":   1.0,
		"status":       "Released",
	}

	// Return the movie info
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(movieInfo)
}

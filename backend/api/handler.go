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

	// Parse the request body
	var req struct {
		InfoHash string          `json:"infoHash"`
		Movie    *db.MovieDetails `json:"movie"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.InfoHash == "" || req.Movie == nil {
		http.Error(w, "Missing infoHash or movie details", http.StatusBadRequest)
		return
	}

	// Check if the torrent exists
	t, ok := h.torrentClient.GetTorrent(req.InfoHash)
	if !ok {
		http.Error(w, "Torrent not found", http.StatusNotFound)
		return
	}

	// Get files info from the torrent client
	clientFiles, err := h.torrentClient.ListFiles(req.InfoHash)
	if err != nil {
		log.Printf("Failed to get files for torrent %s: %v", req.InfoHash, err)
	} else {
		// Convert torrent.FileInfo to db.FileInfo
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
		// Add files to movie details
		req.Movie.Files = dbFiles
	}

	// Get the torrent record from the database
	record, err := h.torrentStore.GetTorrent(req.InfoHash)
	if err != nil {
		// If not found, create a new record
		record = db.TorrentRecord{
			InfoHash:  req.InfoHash,
			Name:      t.Name(), // t.Name is a function
			MagnetURI: "", // We don't have the magnet URI here
			AddedAt:   time.Now(), // t doesn't have AddedAt field
		}
	}

	// Update movie details
	record.MovieDetails = req.Movie
	record.DataPath = "data/" + record.Name // Set data path to the download location

	// Save to database
	if err := h.torrentStore.SaveTorrent(record); err != nil {
		http.Error(w, "Failed to save movie details: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success
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

package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/torrentplayer/backend/api"
	searchhandle "github.com/torrentplayer/backend/api/search"
	"github.com/torrentplayer/backend/backend"
	"github.com/torrentplayer/backend/db"
	"github.com/torrentplayer/backend/torrent"
)

func main() {
	// Load environment variables
	if err := backend.LoadEnv(); err != nil {
		log.Fatalf("Failed to load environment variables: %v", err)
	}
	// Create a data directory for torrent storage
	dataDir := filepath.Join(".", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Initialize torrent client
	torrentClient, err := torrent.NewClient(dataDir)
	if err != nil {
		log.Fatalf("Failed to create torrent client: %v", err)
	}
	defer torrentClient.Close()

	// Initialize torrent store for database operations
	dbPath := filepath.Join(dataDir, "torrents.db")
	torrentStore, err := db.NewTorrentStore(dbPath)
	if err != nil {
		log.Fatalf("Failed to create torrent store: %v", err)
	}
	defer torrentStore.Close()

	// 从数据库恢复种子到torrent client
	log.Println("正在从数据库恢复种子...")
	torrents, err := torrentStore.GetAllTorrents()
	if err != nil {
		log.Printf("从数据库获取种子失败: %v", err)
	} else {
		for _, t := range torrents {
			if t.MagnetURI != "" {
				log.Printf("正在恢复种子: %s", t.Name)
				_, err := torrentClient.AddMagnet(t.MagnetURI)
				if err != nil {
					log.Printf("恢复种子失败 %s: %v", t.InfoHash, err)
				}
			}
		}
		log.Printf("已从数据库恢复 %d 个种子", len(torrents))
	}

	// Setup API handlers
	apiHandler := api.NewHandler(torrentClient, torrentStore)

	// Configure HTTP server
	http.HandleFunc("/magnet/api/magnet", apiHandler.AddMagnet)
	http.HandleFunc("/magnet/api/torrents", apiHandler.ListTorrents)
	http.HandleFunc("/magnet/api/files", apiHandler.ListFiles)
	http.HandleFunc("/magnet/stream/", apiHandler.StreamFile)
	http.HandleFunc("/magnet/search", searchhandle.SearchMovieHandler)
	// Add new endpoints
	http.HandleFunc("/magnet/api/get-movie-details", apiHandler.GetMovieDetails)
	http.HandleFunc("/magnet/api/search", apiHandler.SearchMovie)
	http.HandleFunc("/magnet/api/movie-details/", apiHandler.UpdateMovieDetails)
	http.HandleFunc("/magnet/api/torrents/save-data/", apiHandler.SaveTorrentData)

	// Enable CORS
	http.HandleFunc("/magnet/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		http.Error(w, "Not found", http.StatusNotFound)
	})

	// Start server
	port := "8080"
	log.Printf("Starting server on port %s...", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

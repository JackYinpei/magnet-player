package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/torrentplayer/backend/api"
	"github.com/torrentplayer/backend/torrent"
)

func main() {
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

	// Setup API handlers
	apiHandler := api.NewHandler(torrentClient)

	// Configure HTTP server
	http.HandleFunc("/api/magnet", apiHandler.AddMagnet)
	http.HandleFunc("/api/torrents", apiHandler.ListTorrents)
	http.HandleFunc("/api/files", apiHandler.ListFiles)
	http.HandleFunc("/stream/", apiHandler.StreamFile)
	
	// Enable CORS
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
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

package search

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/torrentplayer/backend/service/search"
)

func SearchMovieHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	filename := r.URL.Query().Get("filename")
	if filename == "" {
		http.Error(w, "Missing filename parameter", http.StatusBadRequest)
		return
	}
	movieInfo, err := search.SearchMovie(filename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Println("根据传进来的magnet filename 通过AI 搜索出的电影信息", movieInfo)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(movieInfo)
}

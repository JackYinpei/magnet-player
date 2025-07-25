package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/torrentplayer/backend/db"
	"github.com/torrentplayer/backend/middleware"
	"github.com/torrentplayer/backend/service"
	"github.com/torrentplayer/backend/validator"
)

// TorrentHandler 种子处理器
type TorrentHandler struct {
	torrentService *service.TorrentService
	searchService  *service.SearchService
}

// NewTorrentHandler 创建种子处理器
func NewTorrentHandler(torrentService *service.TorrentService, searchService *service.SearchService) *TorrentHandler {
	return &TorrentHandler{
		torrentService: torrentService,
		searchService:  searchService,
	}
}

// AddMagnet 添加磁力链接处理器
func (h *TorrentHandler) AddMagnet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MagnetURI string `json:"magnetUri"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteErrorResponse(w, "无效的请求格式", http.StatusBadRequest)
		return
	}

	// 验证磁力链接
	magnetValidator := &validator.MagnetValidator{}
	if err := magnetValidator.ValidateMagnetURI(req.MagnetURI); err != nil {
		middleware.WriteErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 调用服务层
	torrentInfo, err := h.torrentService.AddMagnet(req.MagnetURI)
	if err != nil {
		middleware.WriteErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 返回结果
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(torrentInfo)
}

// ListTorrents 获取种子列表处理器
func (h *TorrentHandler) ListTorrents(w http.ResponseWriter, r *http.Request) {
	torrents, err := h.torrentService.ListTorrents()
	if err != nil {
		middleware.WriteErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(torrents)
}

// UpdateMovieDetails 更新电影详情处理器
func (h *TorrentHandler) UpdateMovieDetails(w http.ResponseWriter, r *http.Request) {
	// 从URL路径中提取InfoHash
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		middleware.WriteErrorResponse(w, "缺少InfoHash参数", http.StatusBadRequest)
		return
	}
	infoHash := pathParts[len(pathParts)-1]

	// 验证InfoHash
	validator := &validator.InfoHashValidator{}
	if err := validator.ValidateInfoHash(infoHash); err != nil {
		middleware.WriteErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 解析请求体
	var movieDetails struct {
		Filename      string   `json:"filename,omitempty"`
		Year          int      `json:"year,omitempty"`
		PosterUrl     string   `json:"posterUrl,omitempty"`
		BackdropUrl   string   `json:"backdropUrl,omitempty"`
		Overview      string   `json:"overview,omitempty"`
		Rating        float64  `json:"rating,omitempty"`
		VoteCount     int      `json:"voteCount,omitempty"`
		Genres        []string `json:"genres,omitempty"`
		Runtime       int      `json:"runtime,omitempty"`
		TmdbId        int      `json:"tmdbId,omitempty"`
		ReleaseDate   string   `json:"releaseDate,omitempty"`
		OriginalTitle string   `json:"originalTitle,omitempty"`
		Popularity    float64  `json:"popularity,omitempty"`
		Status        string   `json:"status,omitempty"`
		Tagline       string   `json:"tagline,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&movieDetails); err != nil {
		middleware.WriteErrorResponse(w, "无效的请求格式", http.StatusBadRequest)
		return
	}

	// 转换为数据库模型
	dbMovieDetails := &db.MovieDetails{
		Filename:      movieDetails.Filename,
		Year:          movieDetails.Year,
		PosterUrl:     movieDetails.PosterUrl,
		BackdropUrl:   movieDetails.BackdropUrl,
		Overview:      movieDetails.Overview,
		Rating:        movieDetails.Rating,
		VoteCount:     movieDetails.VoteCount,
		Genres:        movieDetails.Genres,
		Runtime:       movieDetails.Runtime,
		TmdbId:        movieDetails.TmdbId,
		ReleaseDate:   movieDetails.ReleaseDate,
		OriginalTitle: movieDetails.OriginalTitle,
		Popularity:    movieDetails.Popularity,
		Status:        movieDetails.Status,
		Tagline:       movieDetails.Tagline,
	}

	// 调用服务层
	if err := h.torrentService.UpdateMovieDetails(infoHash, dbMovieDetails); err != nil {
		middleware.WriteErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 返回成功响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// GetMovieDetails 获取电影详情处理器
func (h *TorrentHandler) GetMovieDetails(w http.ResponseWriter, r *http.Request) {
	records, err := h.torrentService.GetMovieDetails()
	if err != nil {
		middleware.WriteErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}

// SaveTorrentData 保存种子数据处理器
func (h *TorrentHandler) SaveTorrentData(w http.ResponseWriter, r *http.Request) {
	// 从URL路径中提取InfoHash
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 5 {
		middleware.WriteErrorResponse(w, "无效的URL路径", http.StatusBadRequest)
		return
	}
	infoHash := pathParts[len(pathParts)-1]

	// 验证InfoHash
	validator := &validator.InfoHashValidator{}
	if err := validator.ValidateInfoHash(infoHash); err != nil {
		middleware.WriteErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 解析请求体
	var torrentData service.TorrentUpdateData
	if err := json.NewDecoder(r.Body).Decode(&torrentData); err != nil {
		middleware.WriteErrorResponse(w, "无效的请求格式", http.StatusBadRequest)
		return
	}

	// 验证InfoHash一致性
	if infoHash != torrentData.InfoHash {
		middleware.WriteErrorResponse(w, "InfoHash不匹配", http.StatusBadRequest)
		return
	}

	// 调用服务层
	if err := h.torrentService.SaveTorrentData(infoHash, &torrentData); err != nil {
		middleware.WriteErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 返回成功响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "种子数据保存成功",
	})
}
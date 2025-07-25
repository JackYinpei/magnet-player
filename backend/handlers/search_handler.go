package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/torrentplayer/backend/middleware"
	"github.com/torrentplayer/backend/service"
	"github.com/torrentplayer/backend/validator"
)

// SearchHandler 搜索处理器
type SearchHandler struct {
	searchService *service.SearchService
}

// NewSearchHandler 创建搜索处理器
func NewSearchHandler(searchService *service.SearchService) *SearchHandler {
	return &SearchHandler{
		searchService: searchService,
	}
}

// SearchMovie 搜索电影处理器
func (h *SearchHandler) SearchMovie(w http.ResponseWriter, r *http.Request) {
	// 获取查询参数
	filename := r.URL.Query().Get("filename")
	if filename == "" {
		middleware.WriteErrorResponse(w, "缺少filename参数", http.StatusBadRequest)
		return
	}

	// 验证输入
	stringValidator := &validator.StringValidator{}
	if err := stringValidator.ValidateRequired(filename, "filename"); err != nil {
		middleware.WriteErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := stringValidator.ValidateMaxLength(filename, "filename", 500); err != nil {
		middleware.WriteErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 调用搜索服务
	movieInfo, err := h.searchService.SearchMovie(filename)
	if err != nil {
		middleware.WriteErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 返回结果
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(movieInfo)
}
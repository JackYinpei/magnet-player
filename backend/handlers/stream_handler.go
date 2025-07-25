package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/torrentplayer/backend/middleware"
	"github.com/torrentplayer/backend/service"
	"github.com/torrentplayer/backend/validator"
)

// StreamHandler 流媒体处理器
type StreamHandler struct {
	torrentService *service.TorrentService
}

// NewStreamHandler 创建流媒体处理器
func NewStreamHandler(torrentService *service.TorrentService) *StreamHandler {
	return &StreamHandler{
		torrentService: torrentService,
	}
}

// StreamFile 流媒体文件处理器
func (h *StreamHandler) StreamFile(w http.ResponseWriter, r *http.Request) {
	// 解析URL路径: /stream/{infoHash}/{fileName}
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		middleware.WriteErrorResponse(w, "无效的URL格式", http.StatusBadRequest)
		return
	}

	infoHash := pathParts[len(pathParts)-2]
	fileName := pathParts[len(pathParts)-1]

	// 验证InfoHash
	ihValidator := &validator.InfoHashValidator{}
	if err := ihValidator.ValidateInfoHash(infoHash); err != nil {
		middleware.WriteErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 验证文件路径
	fpValidator := &validator.FilePathValidator{}
	if err := fpValidator.ValidateFilePath(fileName); err != nil {
		middleware.WriteErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 获取种子信息
	torrentInfo, err := h.torrentService.GetTorrent(infoHash)
	if err != nil {
		middleware.WriteErrorResponse(w, "种子不存在", http.StatusNotFound)
		return
	}

	// 获取文件列表
	filesList, err := h.torrentService.ListFiles(infoHash)
	if err != nil {
		middleware.WriteErrorResponse(w, "获取文件列表失败", http.StatusInternalServerError)
		return
	}

	// 查找匹配的文件
	var fileIndex int = -1
	for _, f := range filesList {
		if f.Path == fileName {
			fileIndex = f.FileIndex
			break
		}
	}

	if fileIndex == -1 {
		middleware.WriteErrorResponse(w, "文件不存在", http.StatusNotFound)
		return
	}

	// 使用原始torrent客户端获取文件流
	// 注意：这里需要访问底层的torrent客户端
	// 在生产环境中，应该在服务层提供流媒体方法
	if err := h.streamFileContent(w, r, infoHash, fileIndex, fileName); err != nil {
		log.Printf("流媒体传输失败: %v", err)
		if !isConnectionClosed(err) {
			middleware.WriteErrorResponse(w, "流媒体传输失败", http.StatusInternalServerError)
		}
	}
}

// streamFileContent 流式传输文件内容
func (h *StreamHandler) streamFileContent(w http.ResponseWriter, r *http.Request, infoHash string, fileIndex int, fileName string) error {
	// 这里需要访问底层的torrent客户端
	// 为了演示，我们假设可以通过服务层获取文件流
	// 在实际实现中，需要在TorrentService中添加GetFileStream方法
	
	// 设置Content-Type
	contentType := getContentTypeFromPath(fileName)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Accept-Ranges", "bytes")

	// 这里应该实现实际的文件流传输逻辑
	// 由于需要访问底层torrent库，暂时返回错误提示
	return fmt.Errorf("流媒体功能需要在服务层实现文件流接口")
}

// getContentTypeFromPath 根据文件路径确定Content-Type
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

// isConnectionClosed 检查连接是否已关闭
func isConnectionClosed(err error) bool {
	if err == nil {
		return false
	}
	
	errStr := err.Error()
	return strings.Contains(errStr, "connection reset") ||
		   strings.Contains(errStr, "broken pipe") ||
		   strings.Contains(errStr, "connection aborted")
}
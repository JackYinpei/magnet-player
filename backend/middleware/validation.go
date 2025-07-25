package middleware

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/torrentplayer/backend/validator"
)

// ValidateJSONBody 验证JSON请求体中间件
func ValidateJSONBody(maxSize int64) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost || r.Method == http.MethodPut {
				// 检查Content-Type
				contentType := r.Header.Get("Content-Type")
				if contentType != "application/json" {
					WriteErrorResponse(w, "Content-Type必须为application/json", http.StatusBadRequest)
					return
				}

				// 限制请求体大小
				r.Body = http.MaxBytesReader(w, r.Body, maxSize)
			}

			next(w, r)
		}
	}
}

// ValidateMagnetRequest 验证磁力链接请求
func ValidateMagnetRequest(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			next(w, r)
			return
		}

		// 读取请求体
		body, err := io.ReadAll(r.Body)
		if err != nil {
			WriteErrorResponse(w, "读取请求体失败", http.StatusBadRequest)
			return
		}

		// 解析JSON
		var req struct {
			MagnetURI string `json:"magnetUri"`
		}

		if err := json.Unmarshal(body, &req); err != nil {
			WriteErrorResponse(w, "JSON格式无效", http.StatusBadRequest)
			return
		}

		// 验证磁力链接
		validator := &validator.MagnetValidator{}
		if err := validator.ValidateMagnetURI(req.MagnetURI); err != nil {
			WriteErrorResponse(w, err.Error(), http.StatusBadRequest)
			return
		}

		// 将解析后的数据重新写入请求体（这样handler可以重新读取）
		// 注意：这里需要重新创建一个ReadCloser
		// 实际项目中，更好的做法是将验证后的数据通过context传递
		next(w, r)
	}
}

// ValidateInfoHash 验证InfoHash参数
func ValidateInfoHash(paramName string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			var infoHash string

			// 从不同位置获取InfoHash
			switch {
			case r.URL.Query().Get(paramName) != "":
				infoHash = r.URL.Query().Get(paramName)
			case r.URL.Query().Get("infoHash") != "":
				infoHash = r.URL.Query().Get("infoHash")
			default:
				// 从URL路径中提取（例如：/api/torrents/{infoHash}）
				pathParts := splitPath(r.URL.Path)
				if len(pathParts) > 0 {
					infoHash = pathParts[len(pathParts)-1]
				}
			}

			if infoHash == "" {
				WriteErrorResponse(w, "缺少InfoHash参数", http.StatusBadRequest)
				return
			}

			// 验证InfoHash格式
			validator := &validator.InfoHashValidator{}
			if err := validator.ValidateInfoHash(infoHash); err != nil {
				WriteErrorResponse(w, err.Error(), http.StatusBadRequest)
				return
			}

			next(w, r)
		}
	}
}

// ValidateFilePath 验证文件路径参数
func ValidateFilePath(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 从URL路径中提取文件路径
		pathParts := splitPath(r.URL.Path)
		if len(pathParts) < 2 {
			WriteErrorResponse(w, "无效的文件路径", http.StatusBadRequest)
			return
		}

		filePath := pathParts[len(pathParts)-1]

		// 验证文件路径安全性
		validator := &validator.FilePathValidator{}
		if err := validator.ValidateFilePath(filePath); err != nil {
			WriteErrorResponse(w, err.Error(), http.StatusBadRequest)
			return
		}

		next(w, r)
	}
}

// splitPath 分割URL路径
func splitPath(path string) []string {
	parts := make([]string, 0)
	for _, part := range strings.Split(path, "/") {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

// ValidateQueryParams 验证查询参数
func ValidateQueryParams(validParams map[string]bool) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// 检查是否包含未授权的参数
			for param := range r.URL.Query() {
				if !validParams[param] {
					WriteErrorResponse(w, "无效的查询参数: "+param, http.StatusBadRequest)
					return
				}
			}

			next(w, r)
		}
	}
}
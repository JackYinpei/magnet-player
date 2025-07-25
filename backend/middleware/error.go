package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime"
)

// ErrorResponse 统一错误响应结构
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code"`
}

// AppError 应用错误类型
type AppError struct {
	Message    string
	StatusCode int
	Internal   error
}

func (e *AppError) Error() string {
	return e.Message
}

// NewAppError 创建新的应用错误
func NewAppError(message string, statusCode int, internal error) *AppError {
	return &AppError{
		Message:    message,
		StatusCode: statusCode,
		Internal:   internal,
	}
}

// ErrorHandler 统一错误处理中间件
func ErrorHandler(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// 获取错误堆栈信息
				buf := make([]byte, 1024)
				stack := runtime.Stack(buf, false)
				log.Printf("Panic recovered: %v\nStack: %s", err, stack)
				
				// 返回500错误
				writeErrorResponse(w, "Internal server error", http.StatusInternalServerError)
			}
		}()
		
		next(w, r)
	}
}

// WriteErrorResponse 写入错误响应
func WriteErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	writeErrorResponse(w, message, statusCode)
}

// writeErrorResponse 内部错误响应写入函数
func writeErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	errorResp := ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
		Code:    statusCode,
	}
	
	if err := json.NewEncoder(w).Encode(errorResp); err != nil {
		log.Printf("Failed to encode error response: %v", err)
	}
}

// ValidateMethod 验证HTTP方法
func ValidateMethod(allowedMethods ...string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			for _, method := range allowedMethods {
				if r.Method == method {
					next(w, r)
					return
				}
			}
			WriteErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}
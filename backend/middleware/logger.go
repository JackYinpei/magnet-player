package middleware

import (
	"log"
	"net/http"
	"time"
)

// Logger 请求日志中间件
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// 创建响应写入器包装器来捕获状态码
		ww := &responseWriter{ResponseWriter: w}
		
		// 处理请求
		next.ServeHTTP(ww, r)
		
		// 记录日志
		duration := time.Since(start)
		log.Printf(
			"%s %s %d %v %s",
			r.Method,
			r.URL.Path,
			ww.statusCode,
			duration,
			r.RemoteAddr,
		)
	})
}

// responseWriter 包装器用于捕获状态码
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}
	return rw.ResponseWriter.Write(b)
}
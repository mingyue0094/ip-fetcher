// main.go - IP Service API
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

const (
	baseURL      = "https://www.cloudflare.com/cdn-cgi/trace"
	userAgent    = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/154.0.0.0 Safari/537.36"
	timeout      = 10 * time.Second
)

// Service 封装获取 IP 的业务逻辑
type Service struct {
	url    string
	headers map[string]string
}

// NewService 创建新的服务实例
func NewService() *Service {
	return &Service{
		url: baseURL,
		headers: map[string]string{
			"Accept":             "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
			"Accept-Language":    "zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7",
			"Cache-Control":      "no-cache",
			"Pragma":             "no-cache",
			"Priority":           "u=0, i",
			"Sec-Ch-Ua":          fmt.Sprintf(`"%s";v="154", "Google Chrome";v="154", "Not A(Brand";v="99"`, userAgent),
			"Sec-Ch-Ua-Mobile":   "?0",
			"Sec-Ch-Ua-Platform": `"Windows"`,
			"Sec-Fetch-Dest":     "document",
			"Sec-Fetch-Mode":     "navigate",
			"Sec-Fetch-Site":     "none",
			"Sec-Fetch-User":     "?1",
			"Upgrade-Insecure-Requests": "1",
			"User-Agent":         userAgent,
		},
	}
}

// getClient 创建带超时的 HTTP 客户端
func getClient() *http.Client {
	return &http.Client{
		Timeout: timeout,
	}
}

// doRequest 执行 HTTP GET 请求并返回响应体
func (s *Service) doRequest(ctx context.Context) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range s.headers {
		req.Header.Set(k, v)
	}

	client := getClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return body, nil
}

// ExtractIP 从响应体中提取 IP 地址
func (s *Service) ExtractIP(body []byte) string {
	lines := strings.Split(string(body), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "ip=") {
			ip := strings.TrimPrefix(line, "ip=")
			return ip
		}
	}
	return ""
}

// GetIP 获取当前 IP 地址
func (s *Service) GetIP(ctx context.Context) string {
	body, err := s.doRequest(ctx)
	if err != nil {
		fmt.Printf("Error fetching IP: %v\n", err)
		return ""
	}
	return s.ExtractIP(body)
}

// Handler 处理 HTTP 请求
type Handler struct {
	service *Service
}

// NewHandler 创建新的处理器实例
func NewHandler(svc *Service) *Handler {
	return &Handler{service: svc}
}

// HandleIP 处理获取 IP 的请求
func (h *Handler) HandleIP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ip := h.service.GetIP(ctx)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	jsonResponse := map[string]interface{}{
		"ip":    ip,
		"status": http.StatusOK,
	}
	json.NewEncoder(w).Encode(jsonResponse)
}

func main() {
	service := NewService()
	handler := NewHandler(service)
	router := chi.NewRouter()

	// 启用 CORS
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// 路由定义
	router.Get("/", handler.HandleIP)
	router.Options("/", handler.HandleIP)
	port := ":8080"
	fmt.Printf("IP Service API started on http://localhost%s\n", port)
	fmt.Println("API Docs: GET /")

	if err := http.ListenAndServe(port, router); err != nil && err != http.ErrServerClosed {
		fmt.Printf("Server error: %v\n", err)
	}
}



// Package oauthlogin 提供 OAuth 登入功能
package oauthlogin

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// CallbackResult 回調結果結構
type CallbackResult struct {
	// Code 授權碼
	Code string
	// State 狀態參數
	State string
}

// CallbackServer 本地回調 Server
type CallbackServer struct {
	expectedState string
	server        *http.Server
	listener      net.Listener
	port          int
	resultChan    chan *CallbackResult
	errorChan     chan error
	mu            sync.Mutex
	stopped       bool
}

// NewCallbackServer 建立新的 Callback Server
func NewCallbackServer(expectedState string) *CallbackServer {
	return &CallbackServer{
		expectedState: expectedState,
		resultChan:    make(chan *CallbackResult, 1),
		errorChan:     make(chan error, 1),
	}
}

// Start 在隨機端口啟動 HTTP Server
// 返回分配的端口號
func (s *CallbackServer) Start() (int, error) {
	// 監聽隨機端口
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	s.listener = listener
	s.port = listener.Addr().(*net.TCPAddr).Port

	// 建立 HTTP Server
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", s.handleCallback)

	s.server = &http.Server{
		Handler: mux,
	}

	// 啟動 Server
	go func() {
		if err := s.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			s.errorChan <- err
		}
	}()

	return s.port, nil
}

// handleCallback 處理 OAuth 回調
func (s *CallbackServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	// 檢查是否有錯誤參數（用戶取消授權）
	if errParam := query.Get("error"); errParam != "" {
		s.errorChan <- &OAuthError{
			Code:    ErrCodeCancelled,
			Message: "用戶取消授權",
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(s.getErrorHTML("授權已取消")))
		return
	}

	// 驗證 state 參數
	state := query.Get("state")
	if !ValidateState(s.expectedState, state) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid state parameter"))
		return
	}

	// 提取 authorization_code
	code := query.Get("code")
	if code == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Missing authorization code"))
		return
	}

	// 發送結果
	s.resultChan <- &CallbackResult{
		Code:  code,
		State: state,
	}

	// 返回成功頁面
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(s.getSuccessHTML()))
}

// WaitForCallback 等待回調結果
// timeout 為等待超時時間
func (s *CallbackServer) WaitForCallback(timeout time.Duration) (*CallbackResult, error) {
	select {
	case result := <-s.resultChan:
		return result, nil
	case err := <-s.errorChan:
		return nil, err
	case <-time.After(timeout):
		return nil, &OAuthError{
			Code:    ErrCodeTimeout,
			Message: "登入超時，請重試",
		}
	}
}

// Stop 關閉 Server
func (s *CallbackServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return nil
	}
	s.stopped = true

	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(ctx)
	}
	return nil
}

// GetCallbackURL 取得回調 URL
func (s *CallbackServer) GetCallbackURL() string {
	return fmt.Sprintf("http://localhost:%d/callback", s.port)
}

// GetPort 取得端口號
func (s *CallbackServer) GetPort() int {
	return s.port
}

// getSuccessHTML 返回成功頁面 HTML
func (s *CallbackServer) getSuccessHTML() string {
	return `<!DOCTYPE html>
<html lang="zh-TW">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>登入成功</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            margin: 0;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        }
        .container {
            text-align: center;
            background: white;
            padding: 40px 60px;
            border-radius: 16px;
            box-shadow: 0 10px 40px rgba(0,0,0,0.2);
        }
        .icon {
            font-size: 64px;
            margin-bottom: 20px;
        }
        h1 {
            color: #333;
            margin-bottom: 10px;
        }
        p {
            color: #666;
            margin-bottom: 20px;
        }
        .hint {
            color: #999;
            font-size: 14px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="icon">✅</div>
        <h1>登入成功</h1>
        <p>您已成功完成授權</p>
        <p class="hint">此視窗將在 5 秒後自動關閉，或您可以直接關閉此視窗</p>
    </div>
    <script>
        setTimeout(function() { window.close(); }, 5000);
    </script>
</body>
</html>`
}

// getErrorHTML 返回錯誤頁面 HTML
func (s *CallbackServer) getErrorHTML(message string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-TW">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>授權失敗</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            margin: 0;
            background: linear-gradient(135deg, #f093fb 0%%, #f5576c 100%%);
        }
        .container {
            text-align: center;
            background: white;
            padding: 40px 60px;
            border-radius: 16px;
            box-shadow: 0 10px 40px rgba(0,0,0,0.2);
        }
        .icon {
            font-size: 64px;
            margin-bottom: 20px;
        }
        h1 {
            color: #333;
            margin-bottom: 10px;
        }
        p {
            color: #666;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="icon">❌</div>
        <h1>授權失敗</h1>
        <p>%s</p>
    </div>
</body>
</html>`, message)
}

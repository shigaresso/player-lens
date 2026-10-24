package adapter

import (
	"bytes"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		status int
		body   string
	}{
		{"GETは正常応答を返す", http.MethodGet, "/healthz", http.StatusOK, "ok\n"},
		{"HEADは本文を返さない", http.MethodHead, "/healthz", http.StatusOK, ""},
		{"POSTは許可しない", http.MethodPost, "/healthz", http.StatusMethodNotAllowed, ""},
		{"未定義パスを公開しない", http.MethodGet, "/", http.StatusNotFound, ""},
		{"子パスをヘルスチェックに含めない", http.MethodGet, "/healthz/child", http.StatusNotFound, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			NewHandler(nil).ServeHTTP(response, httptest.NewRequest(tt.method, tt.path, nil))
			if response.Code != tt.status {
				t.Fatalf("status=%d; want=%d", response.Code, tt.status)
			}
			if tt.status == http.StatusOK {
				if response.Body.String() != tt.body {
					t.Errorf("body=%q; want=%q", response.Body.String(), tt.body)
				}
				if got := response.Header().Get("Content-Type"); got != "text/plain; charset=utf-8" {
					t.Errorf("Content-Type=%q", got)
				}
			}
			if tt.status == http.StatusMethodNotAllowed && response.Header().Get("Allow") != "GET, HEAD" {
				t.Errorf("Allow=%q", response.Header().Get("Allow"))
			}
		})
	}
}

type failedResponse struct{ header http.Header }

func (w failedResponse) Header() http.Header       { return w.header }
func (w failedResponse) WriteHeader(int)           {}
func (w failedResponse) Write([]byte) (int, error) { return 0, errors.New("接続切断") }

func TestHealthWriteFailure(t *testing.T) {
	var output bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&output)
	t.Cleanup(func() { log.SetOutput(previous) })
	NewHandler(nil).ServeHTTP(failedResponse{header: make(http.Header)},
		httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if got := output.String(); !strings.Contains(got, "ヘルスチェック応答") || !strings.Contains(got, "接続切断") {
		t.Fatalf("書き込み失敗の操作と原因が報告されなかった: %q", got)
	}
}

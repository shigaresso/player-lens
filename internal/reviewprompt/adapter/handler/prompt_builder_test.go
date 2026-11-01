package handler

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"html"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func TestPromptBuilderHandler(t *testing.T) {
	handler, err := NewPromptBuilderHandler()
	if err != nil {
		t.Fatal(err)
	}

	for _, method := range []string{http.MethodGet, http.MethodHead} {
		t.Run(method+"で日本語の作成画面を返す", func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(method, "/?private=入力を保存しない", nil))
			if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "text/html; charset=utf-8" {
				t.Fatalf("status=%d headers=%v", response.Code, response.Header())
			}

			if method == http.MethodHead {
				if response.Body.Len() != 0 {
					t.Fatal("HEAD は本文を返さない")
				}

				return
			}

			body := response.Body.String()
			for _, value := range []string{`lang="ja"`, "Steam レビューの分析依頼を作成", `name="steam_url"`, `id="review-options"`, `name="language"`, `name="connection"`, `name="max_reviews"`, `name="page_size"`, `name="max_pages"`, `id="prompt-output"`, "JavaScript"} {
				if !strings.Contains(body, value) {
					t.Errorf("必要な要素がない: %s", value)
				}
			}

			if strings.Contains(body, "入力を保存しない") {
				t.Fatal("query の値をページへ取り込まない")
			}

			attribute := regexp.MustCompile(`data-input-schema="([^"]+)"`).FindStringSubmatch(body)
			if len(attribute) != 2 {
				t.Fatal("画面の入力定義がない")
			}

			var schema struct {
				Properties map[string]struct {
					Maximum float64 `json:"maximum"`
					Default any     `json:"default"`
				} `json:"properties"`
			}
			if err := json.Unmarshal([]byte(html.UnescapeString(attribute[1])), &schema); err != nil {
				t.Fatal(err)
			}

			if schema.Properties["max_reviews"].Maximum != 100 || schema.Properties["max_reviews"].Default != float64(20) || schema.Properties["max_pages"].Maximum != 5 || schema.Properties["page_size"].Default != float64(100) {
				t.Fatalf("MCP の入力条件を共有していない: %+v", schema)
			}

			policy := response.Header().Get("Content-Security-Policy")
			for _, pattern := range []string{`(?s)<script type="module">(.*?)</script>`, `(?s)<style>(.*?)</style>`} {
				content := regexp.MustCompile(pattern).FindStringSubmatch(body)
				if len(content) != 2 {
					t.Fatalf("埋め込んだ画面資産がない: %s", pattern)
				}

				hash := sha256.Sum256([]byte(content[1]))
				if !strings.Contains(policy, "'sha256-"+base64.StdEncoding.EncodeToString(hash[:])+"'") {
					t.Fatal("画面資産と CSP の hash が一致しない")
				}
			}

			if !strings.Contains(policy, "connect-src 'none'") || response.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Fatalf("画面の HTTP ヘッダーが異なる: %v", response.Header())
			}
		})
	}
}

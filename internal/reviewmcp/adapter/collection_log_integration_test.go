//go:build integration

package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shigaresso/player-lens/internal/reviewmcp/adapter/internal/testutil"
	mcpdomain "github.com/shigaresso/player-lens/internal/reviewmcp/domain"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

type collectionLogBuffer struct {
	mu   sync.Mutex
	data bytes.Buffer
}

func (b *collectionLogBuffer) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.data.Write(data)
}

func (b *collectionLogBuffer) snapshot() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()

	return bytes.Clone(b.data.Bytes())
}

func TestMCPCollectionLog(t *testing.T) {
	for _, version := range []string{"2026-07-28", "2025-11-25", "2025-06-18", "2025-03-26"} {
		for _, oversized := range []bool{false, true} {
			name := "通常の収集"
			if oversized {
				name = "本文省略した収集"
			}

			t.Run(version+"で"+name+"と拒否のログが対応する", func(t *testing.T) {
				output := &collectionLogBuffer{}
				logger := slog.New(slog.NewJSONHandler(output, nil)).With("service.name", "player-lens")
				var calls atomic.Int32
				handler, err := NewHandler(testutil.FetchFunc(func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
					calls.Add(1)
					text := "private review"
					if oversized {
						text = strings.Repeat(text, 10000)
					}

					return steamdomain.ReviewPage{Reviews: []steamdomain.Review{{RecommendationID: 1, Text: text}}, NextCursor: "private cursor"}, nil
				}), testutil.Limits(t, mcpdomain.LimitsConfig{RequestsPerMinute: 10, RequestsPerDay: 1, MaxConcurrent: 1}), localHTTPConfig(t), logger, time.Now)
				if err != nil {
					t.Fatal(err)
				}

				server := httptest.NewServer(handler)
				t.Cleanup(server.Close)
				session := connectClient(t, server.URL, version)
				if _, err := session.ListTools(t.Context(), nil); err != nil {
					t.Fatal(err)
				}

				if len(output.snapshot()) != 0 {
					t.Fatal("初期化やツール一覧を収集完了として記録した")
				}

				for i, tc := range []struct {
					name  string
					input string
					code  string
				}{
					{"入力不正", `{"app_id":440,"page_size":"private input"}`, "invalid_input"},
					{"収集", `{"app_id":440,"max_reviews":1}`, ""},
					{"利用枠拒否", `{"app_id":440}`, "daily_limit_reached"},
				} {
					t.Run(tc.name, func(t *testing.T) {
						code := tc.code
						if i == 1 && oversized {
							code = "response_too_large"
						}

						result, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "collect_reviews", Arguments: json.RawMessage(tc.input)})
						if err != nil || result == nil || result.IsError != (code != "") || testutil.DecodeOutput(t, result).ErrorCode != code {
							t.Fatalf("公開応答を変更した: result=%+v err=%v", result, err)
						}

						lines := bytes.Split(bytes.TrimSpace(output.snapshot()), []byte("\n"))
						if len(lines) != i+1 || bytes.Contains(output.snapshot(), []byte("private")) {
							t.Fatal("ログの回数または非漏出の契約が異なる")
						}

						var record map[string]any
						if err := json.Unmarshal(lines[i], &record); err != nil {
							t.Fatal(err)
						}

						if record["event"] != "review_collection.completed" || record["service.name"] != "player-lens" || code != "" && record["error_code"] != code {
							t.Fatalf("ログと公開コードが対応しなかった: %+v", record)
						}

						if i == 1 && record["collected_count"] != float64(1) {
							t.Fatal("省略前の収集件数を記録しなかった")
						}
					})
				}

				if calls.Load() != 1 {
					t.Fatal("拒否時に取得したか不正入力が利用枠を消費した")
				}
			})
		}
	}
}

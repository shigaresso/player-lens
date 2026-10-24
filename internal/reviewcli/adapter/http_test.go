package adapter_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	jsonoutput "github.com/shigaresso/player-lens/internal/jsonoutput/adapter"
	reviewcli "github.com/shigaresso/player-lens/internal/reviewcli/adapter"
	steamadapter "github.com/shigaresso/player-lens/internal/steam/adapter"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestRunWithHTTPClientAndJSONSaver(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		count  int
		reason steamdomain.StopReason
	}{
		{"HTTPからJSONまで複数回取得する", http.StatusOK, 3, steamdomain.StopReviewLimit},
		{"429で部分結果をJSONへ渡す", http.StatusTooManyRequests, 2, steamdomain.StopFailed},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				call := calls.Add(1)
				var input struct {
					Cursor string `json:"cursor"`
					Size   int    `json:"num_per_page"`
				}
				if err := json.Unmarshal([]byte(r.URL.Query().Get("input_json")), &input); err != nil {
					t.Error(err)
					w.WriteHeader(400)
					return
				}
				cursor, size := steamdomain.FirstPageCursor, 2
				body := `{"response":{"reviews":[{"recommendationid":"1","review":"本文1"},{"recommendationid":"2","review":"本文2"}],"cursor":"A"}}`
				if call == 2 {
					cursor, size = "A", 1
					body = `{"response":{"reviews":[{"recommendationid":"3","review":"本文3"}],"cursor":"B"}}`
				}
				if input.Cursor != cursor || input.Size != size {
					t.Errorf("要求入力: %+v", input)
				}
				if call == 2 && test.status != http.StatusOK {
					w.WriteHeader(test.status)
					return
				}
				if _, err := fmt.Fprint(w, body); err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(server.Close)
			httpClient := server.Client()
			t.Cleanup(httpClient.CloseIdleConnections)
			client, err := steamadapter.NewClient(httpClient, server.URL)
			if err != nil {
				t.Fatal(err)
			}
			var output bytes.Buffer
			saver, err := jsonoutput.NewSaver(&output)
			if err != nil {
				t.Fatal(err)
			}
			err = reviewcli.Run(t.Context(), []string{"--app-id=440", "--page-size=2", "--max-reviews=3"}, client, saver)
			if test.status == http.StatusOK && err != nil {
				t.Fatal(err)
			}
			if test.status != http.StatusOK {
				var status *steamadapter.HTTPStatusError
				if !errors.As(err, &status) || status.StatusCode != test.status {
					t.Fatalf("HTTPの原因: %v", err)
				}
			}
			var result struct {
				Reviews []struct {
					RecommendationID string `json:"id"`
				} `json:"reviews"`
				Reason steamdomain.StopReason `json:"stop_reason"`
			}
			if err := json.Unmarshal(output.Bytes(), &result); err != nil || len(result.Reviews) != test.count || result.Reason != test.reason || calls.Load() != 2 {
				t.Fatalf("JSON=%s calls=%d err=%v", output.Bytes(), calls.Load(), err)
			}
		})
	}
}

type memorySaver struct{ result steamdomain.Collection }

func (s *memorySaver) SaveCollection(_ context.Context, result steamdomain.Collection) error {
	s.result = result
	return nil
}

func TestRunWithMemorySaver(t *testing.T) {
	saver := &memorySaver{}
	fetcher := fetcherFunc(func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
		return steamdomain.ReviewPage{}, nil
	})
	if err := reviewcli.Run(t.Context(), []string{"--app-id=440"}, fetcher, saver); err != nil || saver.result.AppID != 440 || saver.result.StopReason != steamdomain.StopExhausted {
		t.Fatalf("JSON実装を使わない保存: %+v err=%v", saver.result, err)
	}
}

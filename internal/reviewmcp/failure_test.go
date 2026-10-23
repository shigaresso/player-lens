package reviewmcp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/shigaresso/player-lens/internal/steam"
)

func TestCollectFailure(t *testing.T) {
	for _, tc := range []struct {
		name    string
		cause   error
		code    string
		partial bool
	}{
		{"途中の取得失敗は部分結果を返す", errors.New("secret: internal URL and credentials"), "fetch_failed", true},
		{"最初の取得失敗は空の部分結果を返す", errors.New("secret: upstream failure"), "fetch_failed", false},
		{"キャンセルは区別して部分結果を返す", context.Canceled, "cancelled", true},
		{"期限超過は区別して部分結果を返す", context.DeadlineExceeded, "deadline_exceeded", true},
		{"カーソル異常も取得失敗とする", steam.ErrCursorCycle, "fetch_failed", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			fetcher := fetchFunc(func(context.Context, steam.AppID, steam.PageOptions) (steam.ReviewPage, error) {
				calls++
				if tc.partial && calls == 1 {
					return steam.ReviewPage{Reviews: []steam.Review{{RecommendationID: 14, Text: "取得済み"}}, NextCursor: "next"}, nil
				}
				return steam.ReviewPage{}, tc.cause
			})
			result, err := collect(context.Background(), fetcher, json.RawMessage(`{"app_id":440,"max_reviews":3}`), newLimiter(DefaultLimits(), time.Now))
			if err != nil || result == nil || !result.IsError {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			if !errors.Is(result.GetError(), tc.cause) {
				t.Fatal("サーバー側で失敗原因を保持しなかった")
			}
			out := decodeOutput(t, result)
			if out.ErrorCode != tc.code || out.Collection.StopReason != steam.StopFailed || out.Message == "" {
				t.Fatal(out)
			}
			want := 0
			if tc.partial {
				want = 1
			}
			if out.ReturnedCount != want || len(out.Collection.Reviews) != want || out.Collection.PagesFetched != want || calls != want+1 {
				t.Fatal(out)
			}
			if tc.partial && out.Collection.Reviews[0].RecommendationID != "14" {
				t.Fatal(out)
			}
			data, marshalErr := json.Marshal(result)
			if marshalErr != nil {
				t.Fatal(marshalErr)
			}
			if strings.Contains(string(data), "secret") {
				t.Fatal("内部の取得エラーを公開した")
			}
		})
	}
}

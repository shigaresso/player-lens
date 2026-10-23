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

func TestLimiterConcurrentAdmission(t *testing.T) {
	gate := newLimiter(Limits{RequestsPerMinute: 100, RequestsPerDay: 100, MaxConcurrent: 3}, time.Now)
	type admission struct {
		release func()
		code    string
	}
	results := make(chan admission, 80)
	start := make(chan struct{})
	for range 80 {
		go func() {
			<-start
			release, code := gate.acquire()
			results <- admission{release, code}
		}()
	}
	close(start)
	var accepted []func()
	for range 80 {
		result := <-results
		if result.code == "" {
			accepted = append(accepted, result.release)
		} else if result.code != "busy" || result.release != nil {
			t.Fatalf("拒否=%+v", result)
		}
	}
	if len(accepted) != 3 {
		t.Fatalf("受付数=%d", len(accepted))
	}
	for _, release := range accepted {
		release()
	}
	release, code := gate.acquire()
	if code != "" {
		t.Fatal(code)
	}
	release()
}

func TestLimiterBusyDoesNotConsumeBudget(t *testing.T) {
	gate := newLimiter(Limits{RequestsPerMinute: 4, RequestsPerDay: 10, MaxConcurrent: 2}, time.Now)
	first, code := gate.acquire()
	if code != "" {
		t.Fatal(code)
	}
	second, code := gate.acquire()
	if code != "" {
		t.Fatal(code)
	}
	if release, code := gate.acquire(); release != nil || code != "busy" {
		t.Fatalf("拒否=%q", code)
	}
	first()
	first()
	third, code := gate.acquire()
	if code != "" {
		t.Fatal("busy が予算を消費した", code)
	}
	if release, code := gate.acquire(); release != nil || code != "busy" {
		t.Fatalf("二重返却で枠が増えた: %q", code)
	}
	second()
	third()
	fourth, code := gate.acquire()
	if code != "" {
		t.Fatal(code)
	}
	defer fourth()
	if release, code := gate.acquire(); release != nil || code != "rate_limited" {
		t.Fatalf("回数制限=%q", code)
	}
}

func TestCollectionReleasesSlotAfterFailure(t *testing.T) {
	for _, tc := range []struct {
		name string
		code string
	}{
		{"途中失敗でも枠を返却し開始回数は戻さない", "fetch_failed"},
		{"実行中のキャンセルでも枠を返却し開始回数は戻さない", "cancelled"},
		{"取得期限でも枠を返却し開始回数は戻さない", "deadline_exceeded"},
		{"応答量超過でも枠を返却し開始回数は戻さない", "response_too_large"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gate := newLimiter(Limits{RequestsPerMinute: 2, RequestsPerDay: 10, MaxConcurrent: 1}, time.Now)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			first := true
			calls := 0
			fetcher := fetchFunc(func(context.Context, uint32, steam.PageOptions) (steam.ReviewPage, error) {
				calls++
				if !first {
					return steam.ReviewPage{}, nil
				}
				switch tc.code {
				case "fetch_failed":
					if calls == 1 {
						return steam.ReviewPage{Reviews: []steam.Review{{RecommendationID: 11}}, NextCursor: "next"}, nil
					}
					return steam.ReviewPage{}, errors.New("fetch failed")
				case "cancelled":
					cancel()
					return steam.ReviewPage{}, ctx.Err()
				case "deadline_exceeded":
					return steam.ReviewPage{}, context.DeadlineExceeded
				default:
					return steam.ReviewPage{Reviews: []steam.Review{{RecommendationID: 10, Text: strings.Repeat("x", 70000)}}, NextCursor: "next"}, nil
				}
			})
			count := 2
			if tc.code == "response_too_large" {
				count = 1
			}
			input, err := json.Marshal(map[string]any{"app_id": 440, "max_reviews": count})
			if err != nil {
				t.Fatal(err)
			}
			result, err := collect(ctx, fetcher, input, gate)
			if err != nil || !result.IsError || decodeOutput(t, result).ErrorCode != tc.code {
				t.Fatalf("初回=%+v %v", result, err)
			}
			first = false
			result, err = collect(context.Background(), fetcher, input, gate)
			if err != nil || result.IsError {
				t.Fatalf("枠を返却しなかった: %+v %v", result, err)
			}
			before := calls
			result, err = collect(context.Background(), fetcher, input, gate)
			if err != nil || !result.IsError || decodeOutput(t, result).ErrorCode != "rate_limited" || calls != before {
				t.Fatalf("開始回数を戻した: %+v %v", result, err)
			}
		})
	}
}

func TestCollectionBusyResponse(t *testing.T) {
	gate := newLimiter(Limits{RequestsPerMinute: 10, RequestsPerDay: 10, MaxConcurrent: 1}, time.Now)
	release, code := gate.acquire()
	if code != "" {
		t.Fatal(code)
	}
	defer release()
	result, err := collect(context.Background(), fetchFunc(func(context.Context, uint32, steam.PageOptions) (steam.ReviewPage, error) {
		t.Fatal("同時枠のない呼び出しで取得した")
		return steam.ReviewPage{}, nil
	}), json.RawMessage(`{"app_id":440}`), gate)
	if err != nil {
		t.Fatal(err)
	}
	out := decodeOutput(t, result)
	if !result.IsError || out.ErrorCode != "busy" || out.Collection.AppID != 440 || out.Collection.StopReason != steam.StopFailed || out.Collection.PagesFetched != 0 || out.ReturnedCount != 0 || len(out.Collection.Reviews) != 0 {
		t.Fatal(out)
	}
}

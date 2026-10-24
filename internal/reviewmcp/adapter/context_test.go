package adapter

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	mcpdomain "github.com/shigaresso/player-lens/internal/reviewmcp/domain"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestCollectionContext(t *testing.T) {
	for _, tc := range []struct {
		name    string
		shorter bool
	}{
		{"既定の8秒期限を設定して終了時に解放する", false},
		{"呼び出し側の短い期限を優先して終了時に解放する", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			type key struct{}
			ctx := context.WithValue(context.Background(), key{}, "request")
			if tc.shorter {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, time.Second)
				defer cancel()
			}
			var passed context.Context
			start := time.Now()
			fetcher := fetchFunc(func(received context.Context, _ steamdomain.AppID, _ steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
				passed = received
				deadline, ok := received.Deadline()
				if !ok || received.Value(key{}) != "request" {
					t.Fatal("期限または値が伝播しなかった")
				}
				if tc.shorter {
					want, _ := ctx.Deadline()
					if !deadline.Equal(want) {
						t.Fatal("短い期限を変更した")
					}
				} else if deadline.Before(start.Add(7*time.Second)) || deadline.After(start.Add(8*time.Second+100*time.Millisecond)) {
					t.Fatalf("既定期限=%v", deadline.Sub(start))
				}
				return steamdomain.ReviewPage{}, nil
			})
			result, err := collect(ctx, fetcher, json.RawMessage(`{"app_id":440}`), mcpdomain.NewLimiter(mcpdomain.DefaultLimits(), time.Now))
			if err != nil || result.IsError {
				t.Fatalf("%+v %v", result, err)
			}
			if passed == nil || passed.Err() != context.Canceled || ctx.Err() != nil {
				t.Fatal("所有する context の解放が不正だった")
			}
		})
	}
	for _, tc := range []struct {
		name    string
		expired bool
		code    string
	}{
		{"開始前のキャンセルでは取得しない", false, "cancelled"},
		{"開始前の期限超過では取得しない", true, "deadline_exceeded"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			if tc.expired {
				cancel()
				ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			} else {
				cancel()
			}
			defer cancel()
			result, err := collect(ctx, fetchFunc(func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
				t.Fatal("終了済み context で取得した")
				return steamdomain.ReviewPage{}, nil
			}), json.RawMessage(`{"app_id":440}`), mcpdomain.NewLimiter(mcpdomain.DefaultLimits(), time.Now))
			if err != nil || !result.IsError || decodeOutput(t, result).ErrorCode != tc.code {
				t.Fatalf("%+v %v", result, err)
			}
		})
	}
}

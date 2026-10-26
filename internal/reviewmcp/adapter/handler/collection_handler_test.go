package handler

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shigaresso/player-lens/internal/reviewmcp/adapter/internal/testutil"
	mcpdomain "github.com/shigaresso/player-lens/internal/reviewmcp/domain"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

// callbackTrackingContext は終了しない親として、context の連携 callback の保持数を観測する。
// AfterFunc は context パッケージが公開する登録・停止の契約を使い、goroutine を起動しない。
type callbackTrackingContext struct {
	context.Context
	done            chan struct{}
	activeCallbacks int
}

func (ctx *callbackTrackingContext) Done() <-chan struct{} {
	return ctx.done
}

func (ctx *callbackTrackingContext) AfterFunc(func()) func() bool {
	ctx.activeCallbacks++
	var once sync.Once
	return func() bool {
		stopped := false
		once.Do(func() {
			ctx.activeCallbacks--
			stopped = true
		})
		return stopped
	}
}

func TestCollectReviewsHandlerCleanup(t *testing.T) {
	fetchErr := errors.New("取得依存の失敗")
	for _, tc := range []struct {
		name      string
		input     string
		fetchErr  error
		wantCode  string
		wantCalls int
	}{
		{"正常終了で親への連携を解除する", `{"app_id":440}`, nil, "", 1},
		{"入力不正の早期終了で親への連携を解除する", `{"app_id":0}`, nil, "invalid_input", 0},
		{"取得失敗でも親への連携を解除する", `{"app_id":440}`, fetchErr, "fetch_failed", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			httpContext := &callbackTrackingContext{Context: context.Background(), done: make(chan struct{})}
			sdkContext := &callbackTrackingContext{Context: context.Background(), done: make(chan struct{})}
			calls := 0
			var passed context.Context
			handler := NewCollectReviewsHandler(testutil.FetchFunc(func(ctx context.Context, _ steamdomain.AppID, _ steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
				calls++
				passed = ctx
				if httpContext.activeCallbacks == 0 || sdkContext.activeCallbacks == 0 {
					t.Error("取得中に親のキャンセルへ連携していない")
				}

				return steamdomain.ReviewPage{}, tc.fetchErr
			}), httpContext, mcpdomain.NewLimiter(mcpdomain.DefaultLimits(), time.Now))
			result, err := handler(sdkContext, &mcp.CallToolRequest{
				Params: &mcp.CallToolParamsRaw{Arguments: json.RawMessage(tc.input)},
			})
			if err != nil || result == nil {
				t.Fatalf("result = %+v, err = %v", result, err)
			}

			if result.IsError != (tc.wantCode != "") || testutil.DecodeOutput(t, result).ErrorCode != tc.wantCode || calls != tc.wantCalls {
				t.Fatalf("result = %+v, calls = %d", result, calls)
			}

			if httpContext.activeCallbacks != 0 || sdkContext.activeCallbacks != 0 {
				t.Fatalf("終了後に連携を保持している: HTTP=%d, SDK=%d", httpContext.activeCallbacks, sdkContext.activeCallbacks)
			}

			if passed != nil && passed.Err() != context.Canceled {
				t.Fatal("取得へ渡した context を解放していない")
			}
		})
	}
}

func TestCollectReviewsHandlerCancellation(t *testing.T) {
	for _, tc := range []struct {
		name       string
		cancelHTTP bool
	}{
		{"HTTP のキャンセルを取得へ伝え SDK の context は維持する", true},
		{"SDK のキャンセルを取得へ伝え HTTP の context は維持する", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			type requestKey struct{}
			httpContext, cancelHTTP := context.WithCancel(context.WithValue(context.Background(), requestKey{}, "request"))
			sdkContext, cancelSDK := context.WithCancel(context.Background())
			started := make(chan context.Context, 1)
			forceStop := make(chan struct{})
			finished := make(chan struct{})
			var result *mcp.CallToolResult
			var handlerErr error
			handler := NewCollectReviewsHandler(testutil.FetchFunc(func(ctx context.Context, _ steamdomain.AppID, _ steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
				started <- ctx
				select {
				case <-ctx.Done():
					return steamdomain.ReviewPage{}, ctx.Err()
				case <-forceStop:
					return steamdomain.ReviewPage{}, errors.New("テストの終了で取得を停止した")
				}
			}), httpContext, mcpdomain.NewLimiter(mcpdomain.DefaultLimits(), time.Now))
			t.Cleanup(func() {
				cancelHTTP()
				cancelSDK()
				close(forceStop)
				select {
				case <-finished:
				case <-time.After(2 * time.Second):
					t.Error("実行 handler が終了しなかった")
				}
			})
			go func() {
				defer close(finished)

				result, handlerErr = handler(sdkContext, &mcp.CallToolRequest{
					Params: &mcp.CallToolParamsRaw{Arguments: json.RawMessage(`{"app_id":440}`)},
				})
			}()
			select {
			case ctx := <-started:
				if ctx.Value(requestKey{}) != "request" {
					t.Fatal("HTTP の context の値が取得へ伝播しなかった")
				}
			case <-time.After(2 * time.Second):
				t.Fatal("取得が開始しなかった")
			}

			if tc.cancelHTTP {
				cancelHTTP()
			} else {
				cancelSDK()
			}

			select {
			case <-finished:
			case <-time.After(2 * time.Second):
				t.Fatal("キャンセルが取得へ伝播しなかった")
			}

			if handlerErr != nil || result == nil || !result.IsError {
				t.Fatalf("result = %+v, err = %v", result, handlerErr)
			}

			if testutil.DecodeOutput(t, result).ErrorCode != "cancelled" || !errors.Is(result.GetError(), context.Canceled) {
				t.Fatalf("キャンセルの結果と原因を保持していない: %+v", result)
			}

			if tc.cancelHTTP && sdkContext.Err() != nil || !tc.cancelHTTP && httpContext.Err() != nil {
				t.Fatal("handler がもう一方の親 context をキャンセルした")
			}
		})
	}
}

func TestCollectReviewsHandlerPanicCleanup(t *testing.T) {
	t.Run("取得中のpanicを伝えてキャンセル連携と同時枠を解放する", func(t *testing.T) {
		httpContext := &callbackTrackingContext{Context: context.Background(), done: make(chan struct{})}
		sdkContext := &callbackTrackingContext{Context: context.Background(), done: make(chan struct{})}
		gate := mcpdomain.NewLimiter(testutil.Limits(t, mcpdomain.LimitsConfig{RequestsPerMinute: 10, RequestsPerDay: 10, MaxConcurrent: 1}), time.Now)
		cause := errors.New("取得依存のpanic")
		var passed context.Context
		handler := NewCollectReviewsHandler(testutil.FetchFunc(func(ctx context.Context, _ steamdomain.AppID, _ steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
			passed = ctx
			if httpContext.activeCallbacks == 0 || sdkContext.activeCallbacks == 0 {
				t.Error("取得中に親のキャンセルへ連携していない")
			}

			panic(cause)
		}), httpContext, gate)
		func() {
			defer func() {
				if got := recover(); got != cause {
					t.Errorf("panicの原因=%v; want=%v", got, cause)
				}
			}()

			_, err := handler(sdkContext, &mcp.CallToolRequest{
				Params: &mcp.CallToolParamsRaw{Arguments: json.RawMessage(`{"app_id":440}`)},
			})
			t.Errorf("panicを呼び出し側へ伝えなかった: err=%v", err)
		}()
		if httpContext.activeCallbacks != 0 || sdkContext.activeCallbacks != 0 {
			t.Fatalf("終了後に連携を保持している: HTTP=%d, SDK=%d", httpContext.activeCallbacks, sdkContext.activeCallbacks)
		}

		if passed == nil || passed.Err() != context.Canceled || httpContext.Err() != nil || sdkContext.Err() != nil {
			t.Fatal("所有するcontextだけを解放していない")
		}

		release, reason := gate.Acquire()
		if release != nil {
			t.Cleanup(release)
		}

		if reason != mcpdomain.NotRejected {
			t.Fatalf("panic後に同時枠を返却しなかった: %v", reason)
		}
	})
}

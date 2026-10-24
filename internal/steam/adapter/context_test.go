package adapter_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	steamadapter "github.com/shigaresso/player-lens/internal/steam/adapter"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestFetchReviewsPageDeadline(t *testing.T) {
	for _, tt := range []struct {
		name          string
		callerTimeout time.Duration
	}{
		{"期限なしでも10秒に制限する", 0},
		{"長い期限を10秒に制限する", time.Minute},
		{"呼び出し側の短い期限を保持する", time.Second},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.callerTimeout != 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, tt.callerTimeout)
				defer cancel()
			}
			var requestContext context.Context
			client, err := steamadapter.NewClient(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				requestContext = r.Context()
				deadline, ok := r.Context().Deadline()
				if !ok {
					t.Error("通信の期限がない")
				} else if tt.callerTimeout == time.Second {
					want, _ := ctx.Deadline()
					if !deadline.Equal(want) {
						t.Errorf("deadline=%v want=%v", deadline, want)
					}
				} else if remaining := time.Until(deadline); remaining <= 0 || remaining > 10*time.Second || remaining < 9*time.Second {
					t.Errorf("remaining=%v", remaining)
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"response":{}}`)), Header: make(http.Header)}, nil
			})}, "")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := client.FetchReviewsPage(ctx, 440, steamdomain.PageOptions{}); err != nil {
				t.Fatal(err)
			}
			if tt.callerTimeout != time.Second && requestContext.Err() != context.Canceled {
				t.Errorf("操作終了後も期限の資源が残った: %v", requestContext.Err())
			}
		})
	}
}

func TestFetchReviewsPageContextFailure(t *testing.T) {
	for _, tt := range []struct {
		name    string
		expired bool
		want    error
	}{
		{"キャンセルを保持する", false, context.Canceled},
		{"期限切れを保持する", true, context.DeadlineExceeded},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var ctx context.Context
			var cancel context.CancelFunc
			if tt.expired {
				ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			} else {
				ctx, cancel = context.WithCancel(context.Background())
				cancel()
			}
			defer cancel()
			calls := 0
			client, err := steamadapter.NewClient(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) { calls++; return nil, r.Context().Err() })}, "")
			if err != nil {
				t.Fatal(err)
			}
			page, err := client.FetchReviewsPage(ctx, 440, steamdomain.PageOptions{})
			if !errors.Is(err, tt.want) || calls > 1 || !reflect.DeepEqual(page, steamdomain.ReviewPage{}) {
				t.Fatalf("page=%#v error=%v calls=%d", page, err, calls)
			}
		})
	}
}

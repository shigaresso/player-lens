package adapter

import (
	"context"
	"encoding/json"
	"io"
	"reflect"
	"testing"
	"time"

	mcpdomain "github.com/shigaresso/player-lens/internal/reviewmcp/domain"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestParseInputReviewType(t *testing.T) {
	for _, tt := range []struct {
		name    string
		input   string
		want    steamdomain.ReviewType
		invalid bool
	}{
		{"省略は全評価", `{"app_id":440}`, steamdomain.ReviewTypeAll, false},
		{"空は全評価", `{"app_id":440,"review_type":""}`, steamdomain.ReviewTypeAll, false},
		{"全評価を明示する", `{"app_id":440,"review_type":"all"}`, steamdomain.ReviewTypeAll, false},
		{"おすすめを指定する", `{"app_id":440,"review_type":"positive"}`, steamdomain.ReviewTypePositive, false},
		{"非おすすめを指定する", `{"app_id":440,"review_type":"negative"}`, steamdomain.ReviewTypeNegative, false},
		{"未知値を拒否する", `{"app_id":440,"review_type":"mixed"}`, 0, true},
		{"大文字を拒否する", `{"app_id":440,"review_type":"Positive"}`, 0, true},
		{"前後空白を拒否する", `{"app_id":440,"review_type":" negative "}`, 0, true},
		{"nullを拒否する", `{"app_id":440,"review_type":null}`, 0, true},
		{"数値を拒否する", `{"app_id":440,"review_type":1}`, 0, true},
		{"配列を拒否する", `{"app_id":440,"review_type":[]}`, 0, true},
		{"真偽値を拒否する", `{"app_id":440,"review_type":true}`, 0, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			id, options, err := parseInput([]byte(tt.input))
			if tt.invalid {
				if err == nil || id != 0 || options != (steamdomain.CollectionOptionsConfig{}) {
					t.Fatalf("id=%d options=%+v err=%v", id, options, err)
				}
			} else if err != nil || id != 440 || options.ReviewType != tt.want {
				t.Fatalf("id=%d options=%+v err=%v", id, options, err)
			}
		})
	}
}

func TestInputSchemaReviewType(t *testing.T) {
	data, err := json.Marshal(inputSchema())
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Required   []string `json:"required"`
		Properties map[string]struct {
			Type        string   `json:"type"`
			Enum        []string `json:"enum"`
			Default     string   `json:"default"`
			Description string   `json:"description"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	field := schema.Properties["review_type"]
	if field.Type != "string" || field.Default != "all" || field.Description == "" || !reflect.DeepEqual(field.Enum, []string{"", "all", "positive", "negative"}) || !reflect.DeepEqual(schema.Required, []string{"app_id"}) {
		t.Fatalf("schema=%s", data)
	}
}

func TestCollectionReviewTypePartialFailure(t *testing.T) {
	calls := 0
	fetcher := fetchFunc(func(_ context.Context, _ steamdomain.AppID, options steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
		calls++
		if options.ReviewType != steamdomain.ReviewTypeNegative {
			t.Fatalf("options=%+v", options)
		}
		if calls == 2 {
			return steamdomain.ReviewPage{}, io.ErrUnexpectedEOF
		}
		return steamdomain.ReviewPage{Reviews: []steamdomain.Review{{RecommendationID: 1, Text: "取得済み"}}, NextCursor: "next"}, nil
	})
	result, err := collect(t.Context(), fetcher, json.RawMessage(`{"app_id":440,"review_type":"negative","page_size":1,"max_reviews":2,"max_pages":2}`), mcpdomain.NewLimiter(mcpdomain.DefaultLimits(), time.Now))
	if err != nil {
		t.Fatal(err)
	}
	out := decodeOutput(t, result)
	if !result.IsError || out.ErrorCode != "fetch_failed" || out.ReturnedCount != 1 || out.Collection.Options.ReviewType != "negative" || out.Collection.PagesFetched != 1 || out.Collection.StopReason != steamdomain.StopFailed || calls != 2 {
		t.Fatalf("out=%+v calls=%d", out, calls)
	}
}

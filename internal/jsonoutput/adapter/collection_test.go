package adapter_test

import (
	"encoding/json"
	"reflect"
	"slices"
	"strconv"
	"testing"

	jsonoutput "github.com/shigaresso/player-lens/internal/jsonoutput/adapter"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestMarshalCollection(t *testing.T) {
	for _, test := range []struct {
		name    string
		reviews []steamdomain.Review
		reason  steamdomain.StopReason
	}{
		{"本文と条件を失わず出力する", []steamdomain.Review{{RecommendationID: 18446744073709551615, Text: "日本語\n\"引用\" <指示>をデータとして保持する", Language: "japanese", Recommended: true, TimestampCreated: 1, TimestampUpdated: 2}}, steamdomain.StopReviewLimit},
		{"nilのレビューも空配列となる", nil, steamdomain.StopExhausted},
		{"部分結果の失敗を出力する", []steamdomain.Review{{RecommendationID: 1, Text: "途中結果"}}, steamdomain.StopFailed},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := steamdomain.Collection{AppID: 440, Options: steamdomain.CollectionOptionsConfig{Language: "japanese", PageSize: 10, MaxReviews: 20, MaxPages: 3}, Reviews: test.reviews, PagesFetched: 2, StopReason: test.reason}
			original := slices.Clone(input.Reviews)
			data, err := jsonoutput.MarshalCollection(input)
			if err != nil {
				t.Fatal(err)
			}

			var result struct {
				Version int               `json:"schema_version"`
				AppID   steamdomain.AppID `json:"app_id"`
				Options struct {
					Language   string `json:"language"`
					PageSize   int    `json:"page_size"`
					MaxReviews int    `json:"max_reviews"`
					MaxPages   int    `json:"max_pages"`
				} `json:"options"`
				Reviews []struct {
					RecommendationID string                       `json:"id"`
					Text             string                       `json:"text"`
					Language         string                       `json:"language"`
					Recommended      bool                         `json:"recommended"`
					TimestampCreated steamdomain.TimestampCreated `json:"created_at"`
					TimestampUpdated steamdomain.TimestampUpdated `json:"updated_at"`
				} `json:"reviews"`
				PagesFetched int    `json:"pages_fetched"`
				StopReason   string `json:"stop_reason"`
			}
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatal(err)
			}

			if result.Version != 1 || result.AppID != 440 || result.Options.Language != "japanese" || result.Options.PageSize != 10 || result.Options.MaxReviews != 20 || result.Options.MaxPages != 3 || result.PagesFetched != 2 || result.StopReason != string(test.reason) || result.Reviews == nil || len(result.Reviews) != len(test.reviews) {
				t.Fatalf("JSONの契約: %s", data)
			}

			for i, review := range test.reviews {
				got := result.Reviews[i]
				if got.RecommendationID != strconv.FormatUint(uint64(review.RecommendationID), 10) || got.Text != review.Text || got.Language != review.Language || got.Recommended != review.Recommended || got.TimestampCreated != review.TimestampCreated || got.TimestampUpdated != review.TimestampUpdated {
					t.Fatalf("レビューを失った: %+v", got)
				}
			}

			if !reflect.DeepEqual(input.Reviews, original) {
				t.Fatal("入力を書き換えた")
			}
		})
	}
}

func TestMarshalCollectionNumericRecommendationID(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   steamdomain.RecommendationID
		want string
	}{
		{"ゼロを文字列で出力する", 0, `"0"`},
		{"uint64最大値を文字列で精度損失なく出力する", 18446744073709551615, `"18446744073709551615"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := jsonoutput.MarshalCollection(steamdomain.Collection{
				AppID:   4294967295,
				Reviews: []steamdomain.Review{{RecommendationID: tc.id, TimestampCreated: 4294967295, TimestampUpdated: 0}},
			})
			if err != nil {
				t.Fatal(err)
			}

			var got struct {
				SchemaVersion int                          `json:"schema_version"`
				AppID         json.RawMessage              `json:"app_id"`
				Reviews       []map[string]json.RawMessage `json:"reviews"`
			}
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatal(err)
			}

			if got.SchemaVersion != 1 || string(got.AppID) != "4294967295" || len(got.Reviews) != 1 {
				t.Fatalf("JSON=%s", data)
			}

			review := got.Reviews[0]
			if len(review) != 6 || string(review["id"]) != tc.want || string(review["created_at"]) != "4294967295" || string(review["updated_at"]) != "0" {
				t.Fatalf("レビューの公開形式を変更した: %s", data)
			}
		})
	}
}

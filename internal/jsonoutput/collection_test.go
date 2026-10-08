package jsonoutput_test

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/shigaresso/player-lens/internal/jsonoutput"
	"github.com/shigaresso/player-lens/internal/steam"
)

func TestMarshalCollection(t *testing.T) {
	for _, test := range []struct {
		name    string
		reviews []steam.Review
		reason  steam.StopReason
	}{
		{"本文と条件を失わず出力する", []steam.Review{{ID: "18446744073709551615", Text: "日本語\n\"引用\" <指示>をデータとして保持する", Language: "japanese", Recommended: true, CreatedAt: 1, UpdatedAt: 2}}, steam.StopReviewLimit},
		{"nilのレビューも空配列となる", nil, steam.StopExhausted},
		{"部分結果の失敗を出力する", []steam.Review{{ID: "1", Text: "途中結果"}}, steam.StopFailed},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := steam.Collection{AppID: 440, Options: steam.CollectionOptions{Language: "japanese", PageSize: 10, MaxReviews: 20, MaxPages: 3}, Reviews: test.reviews, PagesFetched: 2, StopReason: test.reason}
			original := slices.Clone(input.Reviews)
			data, err := jsonoutput.MarshalCollection(input)
			if err != nil {
				t.Fatal(err)
			}
			var result struct {
				Version int    `json:"schema_version"`
				AppID   uint32 `json:"app_id"`
				Options struct {
					Language   string `json:"language"`
					PageSize   int    `json:"page_size"`
					MaxReviews int    `json:"max_reviews"`
					MaxPages   int    `json:"max_pages"`
				} `json:"options"`
				Reviews []struct {
					ID          string `json:"id"`
					Text        string `json:"text"`
					Language    string `json:"language"`
					Recommended bool   `json:"recommended"`
					CreatedAt   uint32 `json:"created_at"`
					UpdatedAt   uint32 `json:"updated_at"`
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
				if got.ID != review.ID || got.Text != review.Text || got.Language != review.Language || got.Recommended != review.Recommended || got.CreatedAt != review.CreatedAt || got.UpdatedAt != review.UpdatedAt {
					t.Fatalf("レビューを失った: %+v", got)
				}
			}
			if !reflect.DeepEqual(input.Reviews, original) {
				t.Fatal("入力を書き換えた")
			}
		})
	}
}

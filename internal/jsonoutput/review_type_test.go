package jsonoutput_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/shigaresso/player-lens/internal/jsonoutput"
	"github.com/shigaresso/player-lens/internal/steam"
)

func TestMarshalCollectionReviewType(t *testing.T) {
	for _, tt := range []struct {
		name    string
		kind    steam.EUserReviewsReviewType
		want    string
		reason  steam.StopReason
		reviews []steam.Review
	}{
		{"全評価の条件を出力する", steam.ReviewTypeAll, "all", steam.StopReviewLimit, []steam.Review{{RecommendationID: 1}}},
		{"おすすめの条件を出力する", steam.ReviewTypePositive, "positive", steam.StopReviewLimit, []steam.Review{{RecommendationID: 2, Recommended: true}}},
		{"非おすすめの空結果でも条件を出力する", steam.ReviewTypeNegative, "negative", steam.StopExhausted, nil},
		{"非おすすめの部分失敗でも条件を出力する", steam.ReviewTypeNegative, "negative", steam.StopFailed, []steam.Review{{RecommendationID: 3}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			input := steam.Collection{AppID: 440, Options: steam.CollectionOptions{ReviewType: tt.kind}, Reviews: tt.reviews, StopReason: tt.reason}
			before := input
			data, err := jsonoutput.MarshalCollection(input)
			if err != nil {
				t.Fatal(err)
			}
			var out struct {
				Version int `json:"schema_version"`
				Options struct {
					ReviewType string `json:"review_type"`
				} `json:"options"`
				StopReason steam.StopReason  `json:"stop_reason"`
				Reviews    []json.RawMessage `json:"reviews"`
			}
			if err := json.Unmarshal(data, &out); err != nil {
				t.Fatal(err)
			}
			if out.Version != 1 || out.Options.ReviewType != tt.want || out.StopReason != tt.reason || out.Reviews == nil || len(out.Reviews) != len(tt.reviews) {
				t.Fatalf("JSON=%s", data)
			}
			if !reflect.DeepEqual(input, before) {
				t.Fatal("入力を変更した")
			}
		})
	}
}

package jsonoutput_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/shigaresso/player-lens/internal/jsonoutput"
	"github.com/shigaresso/player-lens/internal/steam"
)

func TestMarshalCollectionReviewType(t *testing.T) {
	for _, tt := range []struct {
		name    string
		kind    steam.ReviewType
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

func TestMarshalCollectionReviewTypeBoundary(t *testing.T) {
	for _, tc := range []struct {
		name    string
		kind    steam.ReviewType
		invalid bool
	}{
		{"空の結果のゼロ値も全評価の文字列で返す", 0, false},
		{"負の数値ではJSONを返さない", -1, true},
		{"未知の3ではJSONを返さない", 3, true},
		{"int32最大値ではJSONを返さない", 2147483647, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := steam.Collection{Options: steam.CollectionOptions{ReviewType: tc.kind}}
			data, err := jsonoutput.MarshalCollection(input)
			if tc.invalid {
				if data != nil || !errors.Is(err, steam.ErrInvalidArgument) {
					t.Fatalf("JSON=%s error=%v", data, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var got struct {
				Options struct {
					ReviewType string `json:"review_type"`
				} `json:"options"`
			}
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatal(err)
			}
			if got.Options.ReviewType != "all" || input.Options.ReviewType != tc.kind {
				t.Fatalf("JSON=%s input=%+v", data, input)
			}
		})
	}
}

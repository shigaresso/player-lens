package handler

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestResponseSize(t *testing.T) {
	for _, tc := range []struct {
		name   string
		length int
		code   string
	}{
		{"十分に小さい本文は全体を返す", 60000, ""},
		{"ガイド込みで上限を超える本文は省略する", 63000, "response_too_large"},
		{"上限を超える本文は切り詰めず省略を明示する", 70000, "response_too_large"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			collection := steamdomain.Collection{AppID: 440, Options: steamdomain.CollectionOptionsConfig{Language: "english", PageSize: 100, MaxReviews: 20, MaxPages: 3}, Reviews: []steamdomain.Review{{RecommendationID: 10, Text: strings.Repeat("x", tc.length)}}, PagesFetched: 1, StopReason: steamdomain.StopReviewLimit}
			result, err := encodeResult(collection, toolFailureInfo{})
			if err != nil {
				t.Fatal(err)
			}

			data, err := json.Marshal(result)
			if err != nil {
				t.Fatal(err)
			}

			if len(data) > 128*1024 {
				t.Fatalf("応答量 %d が上限を超えた", len(data))
			}

			out := decodeOutput(t, result)
			if out.ErrorCode != tc.code || result.IsError != (tc.code != "") {
				t.Fatal(out)
			}

			if out.Collection.AppID != 440 || out.Collection.Options.MaxReviews != 20 || out.Collection.PagesFetched != 1 || out.Collection.StopReason != steamdomain.StopReviewLimit {
				t.Fatal(out)
			}

			if tc.code == "" {
				if out.ReturnedCount != 1 || len(out.Collection.Reviews) != 1 || out.Collection.Reviews[0].Text != collection.Reviews[0].Text || out.OmittedReviews != 0 {
					t.Fatal("本文を保持しなかった")
				}
			} else if out.ReturnedCount != 0 || len(out.Collection.Reviews) != 0 || out.OmittedReviews != 1 || out.Message == "" {
				t.Fatal(out)
			}

			if len(collection.Reviews) != 1 || len(collection.Reviews[0].Text) != tc.length {
				t.Fatal("入力を変更した")
			}
		})
	}
}

func TestEncodeResultRejectsOversizedMetadata(t *testing.T) {
	for _, tc := range []struct {
		name    string
		reviews []steamdomain.Review
	}{
		{"本文が最初からなくても大きいメタデータを拒否して終了する", nil},
		{"本文を省略しても大きいメタデータを拒否して終了する", []steamdomain.Review{{RecommendationID: 1, Text: "保持する本文"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			collection := steamdomain.Collection{
				AppID:   440,
				Options: steamdomain.CollectionOptionsConfig{Language: strings.Repeat("x", maxResultBytes+1)},
				Reviews: tc.reviews,
			}
			result, err := encodeResult(collection, toolFailureInfo{})
			if result != nil || !errors.Is(err, errResponseTooLarge) {
				t.Fatalf("result=%+v error=%v", result, err)
			}
		})
	}
}

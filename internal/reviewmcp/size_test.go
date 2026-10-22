package reviewmcp

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/shigaresso/player-lens/internal/steam"
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
			collection := steam.Collection{AppID: 440, Options: steam.CollectionOptions{Language: "english", PageSize: 100, MaxReviews: 20, MaxPages: 3}, Reviews: []steam.Review{{ID: "large", Text: strings.Repeat("x", tc.length)}}, PagesFetched: 1, StopReason: steam.StopReviewLimit}
			result, err := encodeResult(collection, "", "", 0)
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
			if out.Collection.AppID != 440 || out.Collection.Options.MaxReviews != 20 || out.Collection.PagesFetched != 1 || out.Collection.StopReason != steam.StopReviewLimit {
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

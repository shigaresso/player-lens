package adapter

import (
	"errors"
	"math"
	"strconv"
	"testing"

	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestConvertReviewSuccess(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input reviewPageReview
		want  steamdomain.Review
	}{
		{
			name: "全項目を保持し先頭ゼロを数値へ正規化する",
			input: reviewPageReview{
				RecommendationID: "0001", Text: "日本語\n\"引用\"", Language: "japanese",
				Recommended: false, TimestampCreated: 0, TimestampUpdated: math.MaxUint32,
			},
			want: steamdomain.Review{
				RecommendationID: 1, Text: "日本語\n\"引用\"", Language: "japanese",
				Recommended: false, TimestampCreated: 0, TimestampUpdated: math.MaxUint32,
			},
		},
		{
			name: "uint64最大のIDと推薦と作成時刻を保持する",
			input: reviewPageReview{
				RecommendationID: "18446744073709551615", Text: "review text", Language: "english",
				Recommended: true, TimestampCreated: math.MaxUint32, TimestampUpdated: 1,
			},
			want: steamdomain.Review{
				RecommendationID: math.MaxUint64, Text: "review text", Language: "english",
				Recommended: true, TimestampCreated: math.MaxUint32, TimestampUpdated: 1,
			},
		},
		{
			name:  "IDの0と任意項目のゼロ値を保持する",
			input: reviewPageReview{RecommendationID: "0"},
			want:  steamdomain.Review{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := convertReview(tc.input)
			if err != nil || got != tc.want {
				t.Fatalf("review=%+v err=%v; want=%+v", got, err, tc.want)
			}
		})
	}
}

func TestConvertReviewInvalidID(t *testing.T) {
	for _, tc := range []struct {
		name  string
		id    string
		cause error
	}{
		{"空のIDではゼロ値と構文の原因を返す", "", strconv.ErrSyntax},
		{"不正なIDではゼロ値と構文の原因を返す", "bad", strconv.ErrSyntax},
		{"範囲超過ではゼロ値と範囲の原因を返す", "18446744073709551616", strconv.ErrRange},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := convertReview(reviewPageReview{
				RecommendationID: tc.id, Text: "返さない本文", Language: "japanese",
				Recommended: true, TimestampCreated: 10, TimestampUpdated: 20,
			})
			if got != (steamdomain.Review{}) || !errors.Is(err, ErrInvalidResponse) || !errors.Is(err, tc.cause) {
				t.Fatalf("review=%+v err=%v", got, err)
			}

			var numberError *strconv.NumError
			if !errors.As(err, &numberError) || numberError.Num != tc.id {
				t.Fatalf("数値変換の入力と原因を保持しなかった: %v", err)
			}
		})
	}
}

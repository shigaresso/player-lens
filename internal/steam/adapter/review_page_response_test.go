package adapter

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestParseRecommendationID(t *testing.T) {
	for _, tc := range []struct {
		name    string
		value   string
		want    steamdomain.RecommendationID
		wantErr error
	}{
		{"ゼロを受け付ける", "0", 0, nil},
		{"最小の正数を受け付ける", "1", 1, nil},
		{"uint64の最大値を保持する", "18446744073709551615", 18446744073709551615, nil},
		{"先頭ゼロを数値に正規化する", "0001", 1, nil},
		{"ゼロだけの数字列を受け付ける", "000", 0, nil},
		{"空文字を拒否する", "", 0, strconv.ErrSyntax},
		{"負の符号を拒否する", "-1", 0, strconv.ErrSyntax},
		{"正の符号を拒否する", "+1", 0, strconv.ErrSyntax},
		{"uint64の範囲超過ではゼロ値を返す", "18446744073709551616", 0, strconv.ErrRange},
		{"小数を拒否する", "1.0", 0, strconv.ErrSyntax},
		{"指数表記を拒否する", "1e2", 0, strconv.ErrSyntax},
		{"十六進表記を拒否する", "0x10", 0, strconv.ErrSyntax},
		{"数字以外を含む値を拒否する", "1a", 0, strconv.ErrSyntax},
		{"先頭の空白を拒否する", " 1", 0, strconv.ErrSyntax},
		{"末尾の空白を拒否する", "1 ", 0, strconv.ErrSyntax},
		{"末尾の改行を拒否する", "1\n", 0, strconv.ErrSyntax},
		{"全角の数字を拒否する", "１", 0, strconv.ErrSyntax},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseRecommendationID(tc.value)
			if got != tc.want || !errors.Is(err, tc.wantErr) {
				t.Fatalf("id=%d error=%v want=%d wantError=%v", got, err, tc.want, tc.wantErr)
			}

			if tc.wantErr != nil {
				var cause *strconv.NumError
				if !errors.As(err, &cause) {
					t.Fatalf("数値変換の原因を失った: %v", err)
				}
			}
		})
	}
}

func TestDecodeReviewPageRecommendationIDErrorCause(t *testing.T) {
	for _, tc := range []struct {
		name, value  string
		priorReviews string
		wantReview   int
		wantErr      error
	}{
		{"1件目の空のIDでは位置と構文エラーを保持する", "", "", 1, strconv.ErrSyntax},
		{"2件目の空のIDでは位置と構文エラーを保持する", "", `{"recommendationid":"1"},`, 2, strconv.ErrSyntax},
		{"2件目の不正なIDでは位置と構文エラーを保持する", "bad", `{"recommendationid":"1"},`, 2, strconv.ErrSyntax},
		{"2件目のuint64の範囲超過では位置と原因を保持する", "18446744073709551616", `{"recommendationid":"1"},`, 2, strconv.ErrRange},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := fmt.Sprintf(`{"response":{"reviews":[%s{"recommendationid":%q}],"cursor":"next"}}`, tc.priorReviews, tc.value)
			got, err := decodeReviewPage(440, steamdomain.PageOptions{PageSize: 2}, []byte(body))
			if !errors.Is(err, ErrInvalidResponse) || !errors.Is(err, tc.wantErr) || !reflect.DeepEqual(got, steamdomain.ReviewPage{}) {
				t.Fatalf("page=%#v error=%v", got, err)
			}

			var cause *strconv.NumError
			if !errors.As(err, &cause) || cause.Num != tc.value {
				t.Fatalf("数値変換の原因を失った: %v", err)
			}

			if wantContext := fmt.Sprintf("convert Steam review %d:", tc.wantReview); !strings.HasPrefix(err.Error(), wantContext) {
				t.Fatalf("失敗位置=%v wantContext=%q", err, wantContext)
			}
		})
	}
}

func TestDecodeReviewPageSuccess(t *testing.T) {
	options := steamdomain.PageOptions{ReviewType: steamdomain.ReviewTypeAll, Language: "japanese", Cursor: "取得元+/=", PageSize: 2}
	for _, tc := range []struct {
		name       string
		body       string
		reviews    []steamdomain.Review
		nextCursor string
	}{
		{
			name: "要求件数ちょうどのレビューと条件と順序を保持し未知フィールドを無視する",
			body: `{"unknown":true,"response":{"unknown":42,"reviews":[
				{"recommendationid":"18446744073709551615","review":"面白いゲーム","language":"japanese","voted_up":true,"timestamp_created":1,"timestamp_updated":4294967295,"unknown":"value"},
				{"recommendationid":"0001","review":"改善してほしい","language":"english","voted_up":false,"timestamp_created":0,"timestamp_updated":2}
			],"cursor":"次+/=&?"}}`,
			reviews: []steamdomain.Review{
				{RecommendationID: 18446744073709551615, Text: "面白いゲーム", Language: "japanese", Recommended: true, TimestampCreated: 1, TimestampUpdated: 4294967295},
				{RecommendationID: 1, Text: "改善してほしい", Language: "english", Recommended: false, TimestampCreated: 0, TimestampUpdated: 2},
			},
			nextCursor: "次+/=&?",
		},
		{name: "ゼロIDを数値として受け付ける", body: `{"response":{"reviews":[{"recommendationid":"0"}],"cursor":"next"}}`, reviews: []steamdomain.Review{{RecommendationID: steamdomain.RecommendationID(0)}}, nextCursor: "next"},
		{name: "空配列と省略されたカーソルを受け付ける", body: `{"response":{"reviews":[]}}`, reviews: []steamdomain.Review{}},
		{name: "nullのレビューを空配列へ正規化する", body: `{"response":{"reviews":null}}`, reviews: []steamdomain.Review{}},
		{name: "省略されたレビューを空配列へ正規化する", body: `{"response":{}}`, reviews: []steamdomain.Review{}},
		{name: "空ページにあるカーソルを保持する", body: `{"response":{"reviews":[],"cursor":"続き"}}`, reviews: []steamdomain.Review{}, nextCursor: "続き"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := decodeReviewPage(570, options, []byte(tc.body))
			want := steamdomain.ReviewPage{AppID: 570, Options: options, Reviews: tc.reviews, NextCursor: tc.nextCursor}
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("page=%#v error=%v want=%#v", got, err, want)
			}
		})
	}
}

func TestDecodeReviewPageInvalidResponse(t *testing.T) {
	for _, tc := range []struct {
		name, body  string
		pageSize    int
		wantContext string
	}{
		{"wrapper欠落では欠落の説明とゼロ値と不正応答エラーを返す", `{}`, 2, "missing Steam review response"},
		{"ID欠落をゼロと区別して拒否する", `{"response":{"reviews":[{}],"cursor":"next"}}`, 2, ""},
		{"nullのIDをゼロと区別して拒否する", `{"response":{"reviews":[{"recommendationid":null}],"cursor":"next"}}`, 2, ""},
		{"nullの本文では欠落の説明とゼロ値と不正応答エラーを返す", `null`, 2, "missing Steam review response"},
		{"nullのwrapperでは欠落の説明とゼロ値と不正応答エラーを返す", `{"response":null}`, 2, "missing Steam review response"},
		{"要求2件に対する3件の受信では件数を含む説明とゼロ値と不正応答エラーを返す", `{"response":{"reviews":[{"recommendationid":"1"},{"recommendationid":"2"},{"recommendationid":"3"}],"cursor":"next"}}`, 2, "Steam review count 3 exceeds requested page size 2"},
		{"要求20件に対する21件の受信では件数を含む説明とゼロ値と不正応答エラーを返す", `{"response":{"reviews":[` + strings.Repeat(`{"recommendationid":"1"},`, 20) + `{"recommendationid":"1"}],"cursor":"next"}}`, 20, "Steam review count 21 exceeds requested page size 20"},
		{"非空ページのカーソル欠落ではゼロ値と不正応答エラーを返す", `{"response":{"reviews":[{"recommendationid":"1"}]}}`, 2, ""},
		{"非空ページの空カーソルではゼロ値と不正応答エラーを返す", `{"response":{"reviews":[{"recommendationid":"1"}],"cursor":""}}`, 2, ""},
		{"不正IDではゼロ値と不正応答エラーを返す", `{"response":{"reviews":[{"recommendationid":""}],"cursor":"next"}}`, 2, ""},
		{"後続レビューのIDが不正なら先行レビューも返さない", `{"response":{"reviews":[{"recommendationid":"1"},{"recommendationid":"bad"}],"cursor":"next"}}`, 2, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			options := steamdomain.PageOptions{ReviewType: steamdomain.ReviewTypeAll, Language: "english", Cursor: steamdomain.FirstPageCursor, PageSize: tc.pageSize}
			got, err := decodeReviewPage(440, options, []byte(tc.body))
			if !errors.Is(err, ErrInvalidResponse) || !reflect.DeepEqual(got, steamdomain.ReviewPage{}) {
				t.Fatalf("page=%#v error=%v", got, err)
			}

			if tc.wantContext != "" && !strings.HasPrefix(err.Error(), tc.wantContext+":") {
				t.Fatalf("失敗の説明=%v wantContext=%q", err, tc.wantContext)
			}
		})
	}
}

func TestDecodeReviewPageJSONErrorCause(t *testing.T) {
	options := steamdomain.PageOptions{ReviewType: steamdomain.ReviewTypeAll, Language: "english", Cursor: steamdomain.FirstPageCursor, PageSize: 2}
	for _, tc := range []struct {
		name   string
		body   string
		syntax bool
	}{
		{"空の本文ではJSON構文の原因を保持する", "", true},
		{"途中で切れたJSONでは構文の原因を保持する", `{"response":`, true},
		{"後続JSONを拒否し構文の原因を保持する", `{"response":{}}{}`, true},
		{"wrapperが配列なら型の原因を保持する", `{"response":[]}`, false},
		{"reviewsがオブジェクトなら型の原因を保持する", `{"response":{"reviews":{}}}`, false},
		{"IDが数値なら型の原因を保持する", `{"response":{"reviews":[{"recommendationid":1}],"cursor":"next"}}`, false},
		{"作成時刻が文字列なら型の原因を保持する", `{"response":{"reviews":[{"recommendationid":"1","timestamp_created":"100"}],"cursor":"next"}}`, false},
		{"更新時刻がuint32の範囲外なら型の原因を保持する", `{"response":{"reviews":[{"recommendationid":"1","timestamp_updated":4294967296}],"cursor":"next"}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := decodeReviewPage(440, options, []byte(tc.body))
			if !errors.Is(err, ErrInvalidResponse) || !reflect.DeepEqual(got, steamdomain.ReviewPage{}) {
				t.Fatalf("page=%#v error=%v", got, err)
			}

			var syntaxError *json.SyntaxError
			var typeError *json.UnmarshalTypeError
			if errors.As(err, &syntaxError) != tc.syntax || errors.As(err, &typeError) == tc.syntax {
				t.Fatalf("JSONの原因を保持していない: error=%v wantSyntaxError=%v", err, tc.syntax)
			}
		})
	}
}

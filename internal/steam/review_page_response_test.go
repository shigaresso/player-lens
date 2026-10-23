package steam

import (
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"testing"
)

func TestParseRecommendationID(t *testing.T) {
	for _, tc := range []struct {
		name    string
		value   string
		want    RecommendationID
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

func TestDecodeReviewPageSuccess(t *testing.T) {
	options := PageOptions{ReviewType: ReviewTypeAll, Language: "japanese", Cursor: "取得元+/=", PageSize: 2}
	for _, tc := range []struct {
		name       string
		body       string
		reviews    []Review
		nextCursor string
	}{
		{
			name: "要求件数ちょうどのレビューと条件と順序を保持し未知フィールドを無視する",
			body: `{"unknown":true,"response":{"unknown":42,"reviews":[
				{"recommendationid":"18446744073709551615","review":"面白いゲーム","language":"japanese","voted_up":true,"timestamp_created":1,"timestamp_updated":4294967295,"unknown":"value"},
				{"recommendationid":"0001","review":"改善してほしい","language":"english","voted_up":false,"timestamp_created":0,"timestamp_updated":2}
			],"cursor":"次+/=&?"}}`,
			reviews: []Review{
				{RecommendationID: 18446744073709551615, Text: "面白いゲーム", Language: "japanese", Recommended: true, TimestampCreated: 1, TimestampUpdated: 4294967295},
				{RecommendationID: 1, Text: "改善してほしい", Language: "english", Recommended: false, TimestampCreated: 0, TimestampUpdated: 2},
			},
			nextCursor: "次+/=&?",
		},
		{name: "ゼロIDを数値として受け付ける", body: `{"response":{"reviews":[{"recommendationid":"0"}],"cursor":"next"}}`, reviews: []Review{{RecommendationID: RecommendationID(0)}}, nextCursor: "next"},
		{name: "空配列と省略されたカーソルを受け付ける", body: `{"response":{"reviews":[]}}`, reviews: []Review{}},
		{name: "nullのレビューを空配列へ正規化する", body: `{"response":{"reviews":null}}`, reviews: []Review{}},
		{name: "省略されたレビューを空配列へ正規化する", body: `{"response":{}}`, reviews: []Review{}},
		{name: "空ページにあるカーソルを保持する", body: `{"response":{"reviews":[],"cursor":"続き"}}`, reviews: []Review{}, nextCursor: "続き"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := decodeReviewPage(570, options, []byte(tc.body))
			want := ReviewPage{AppID: 570, Options: options, Reviews: tc.reviews, NextCursor: tc.nextCursor}
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("page=%#v error=%v want=%#v", got, err, want)
			}
		})
	}
}

func TestDecodeReviewPageInvalidResponse(t *testing.T) {
	options := PageOptions{ReviewType: ReviewTypeAll, Language: "english", Cursor: FirstPageCursor, PageSize: 2}
	for _, tc := range []struct{ name, body string }{
		{"wrapper欠落ではゼロ値と不正応答エラーを返す", `{}`},
		{"ID欠落をゼロと区別して拒否する", `{"response":{"reviews":[{}],"cursor":"next"}}`},
		{"nullのIDをゼロと区別して拒否する", `{"response":{"reviews":[{"recommendationid":null}],"cursor":"next"}}`},
		{"nullの本文ではゼロ値と不正応答エラーを返す", `null`},
		{"nullのwrapperではゼロ値と不正応答エラーを返す", `{"response":null}`},
		{"要求件数超過ではゼロ値と不正応答エラーを返す", `{"response":{"reviews":[{"recommendationid":"1"},{"recommendationid":"2"},{"recommendationid":"3"}],"cursor":"next"}}`},
		{"非空ページのカーソル欠落ではゼロ値と不正応答エラーを返す", `{"response":{"reviews":[{"recommendationid":"1"}]}}`},
		{"非空ページの空カーソルではゼロ値と不正応答エラーを返す", `{"response":{"reviews":[{"recommendationid":"1"}],"cursor":""}}`},
		{"不正IDではゼロ値と不正応答エラーを返す", `{"response":{"reviews":[{"recommendationid":""}],"cursor":"next"}}`},
		{"後続レビューのIDが不正なら先行レビューも返さない", `{"response":{"reviews":[{"recommendationid":"1"},{"recommendationid":"bad"}],"cursor":"next"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := decodeReviewPage(440, options, []byte(tc.body))
			if !errors.Is(err, ErrInvalidResponse) || !reflect.DeepEqual(got, ReviewPage{}) {
				t.Fatalf("page=%#v error=%v", got, err)
			}
		})
	}
}

func TestDecodeReviewPageJSONErrorCause(t *testing.T) {
	options := PageOptions{ReviewType: ReviewTypeAll, Language: "english", Cursor: FirstPageCursor, PageSize: 2}
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
			if !errors.Is(err, ErrInvalidResponse) || !reflect.DeepEqual(got, ReviewPage{}) {
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

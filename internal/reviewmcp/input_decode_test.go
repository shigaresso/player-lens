package reviewmcp

import (
	"errors"
	"testing"

	"github.com/shigaresso/player-lens/internal/steam"
)

func TestDecodeToolInput(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input string
		want  toolInput
	}{
		{"省略項目へ既定値を入れずに読み取る", `{"app_id":440}`, toolInput{AppID: 440}},
		{"空オブジェクトの条件検査は後段へ渡す", `{}`, toolInput{}},
		{"全項目と App ID の最大値を読み取る", `{"app_id":4294967295,"review_type":"negative","language":"japanese","page_size":100,"max_reviews":100,"max_pages":5}`, toolInput{AppID: 4294967295, ReviewType: "negative", Language: "japanese", PageSize: 100, MaxReviews: 100, MaxPages: 5}},
		{"形式上読める値は条件を正規化せずに保持する", `{"app_id":0,"review_type":"MIXED","language":"日本語","page_size":-1,"max_reviews":101,"max_pages":6}`, toolInput{ReviewType: "MIXED", Language: "日本語", PageSize: -1, MaxReviews: 101, MaxPages: 6}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := decodeToolInput([]byte(tc.input))
			if err != nil || got != tc.want {
				t.Fatalf("input = %+v, err = %v; want %+v", got, err, tc.want)
			}
		})
	}
	for _, tc := range []struct {
		name  string
		input string
	}{
		{"空の入力", ""},
		{"null オブジェクト", `null`},
		{"配列", `[]`},
		{"数値の値", `440`},
		{"壊れた JSON", `{"app_id":`},
		{"後ろに追加の JSON", `{"app_id":440} {}`},
		{"未知の項目", `{"app_id":440,"cursor":"*"}`},
		{"大文字の項目名", `{"app_id":440,"MAX_PAGES":1}`},
		{"App ID が null", `{"app_id":null}`},
		{"評価種別が null", `{"app_id":440,"review_type":null}`},
		{"言語が null", `{"app_id":440,"language":null}`},
		{"ページ件数が null", `{"app_id":440,"page_size":null}`},
		{"収集件数が null", `{"app_id":440,"max_reviews":null}`},
		{"ページ数が null", `{"app_id":440,"max_pages":null}`},
		{"型が異なる項目を途中で読む", `{"app_id":440,"page_size":"1"}`},
		{"文字列の App ID", `{"app_id":"440"}`},
		{"負の App ID", `{"app_id":-1}`},
		{"App ID の型の上限を超える", `{"app_id":4294967296}`},
		{"int の範囲を超える件数", `{"app_id":440,"max_reviews":9223372036854775808}`},
	} {
		t.Run(tc.name+"をゼロ値と原因付きエラーで拒否する", func(t *testing.T) {
			got, err := decodeToolInput([]byte(tc.input))
			if !errors.Is(err, steam.ErrInvalidArgument) || got != (toolInput{}) {
				t.Fatalf("input = %+v, err = %v", got, err)
			}
		})
	}
}

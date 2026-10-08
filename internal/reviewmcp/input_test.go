package reviewmcp

import (
	"testing"

	"github.com/shigaresso/player-lens/internal/steam"
)

func TestParseInput(t *testing.T) {
	for _, tc := range []struct {
		name    string
		input   string
		appID   uint32
		options steam.CollectionOptions
	}{
		{"省略時は少量の既定値となる", `{"app_id":440}`, 440, steam.CollectionOptions{Language: "english", PageSize: 100, MaxReviews: 20, MaxPages: 3}},
		{"ゼロ値と空言語は既定値となる", `{"app_id":440,"language":"","page_size":0,"max_reviews":0,"max_pages":0}`, 440, steam.CollectionOptions{Language: "english", PageSize: 100, MaxReviews: 20, MaxPages: 3}},
		{"上限と有効な言語を受け付ける", `{"app_id":4294967295,"language":"japanese","page_size":100,"max_reviews":100,"max_pages":5}`, 4294967295, steam.CollectionOptions{Language: "japanese", PageSize: 100, MaxReviews: 100, MaxPages: 5}},
		{"最小の取得条件を受け付ける", `{"app_id":1,"language":"all","page_size":1,"max_reviews":1,"max_pages":1}`, 1, steam.CollectionOptions{Language: "all", PageSize: 1, MaxReviews: 1, MaxPages: 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			appID, options, err := parseInput([]byte(tc.input))
			if err != nil || appID != tc.appID || options != tc.options {
				t.Fatalf("parseInput = %d, %+v, %v", appID, options, err)
			}
		})
	}
	for name, input := range map[string]string{
		"AppIDが欠落した": `{}`, "AppIDがゼロだった": `{"app_id":0}`,
		"AppIDが負数だった": `{"app_id":-1}`, "AppIDが上限を超えた": `{"app_id":4294967296}`,
		"AppIDが小数だった": `{"app_id":1.5}`, "AppIDが文字列だった": `{"app_id":"440"}`,
		"件数が負数だった": `{"app_id":440,"max_reviews":-1}`, "件数が上限を超えた": `{"app_id":440,"max_reviews":101}`,
		"ページ数が負数だった": `{"app_id":440,"max_pages":-1}`, "ページ数が上限を超えた": `{"app_id":440,"max_pages":6}`,
		"ページ件数が負数だった": `{"app_id":440,"page_size":-1}`, "ページ件数が上限を超えた": `{"app_id":440,"page_size":101}`,
		"言語が不正だった": `{"app_id":440,"language":"日本語"}`, "未知の項目があった": `{"app_id":440,"cursor":"*"}`,
		"nullだった": `null`, "配列だった": `[]`, "壊れたJSONだった": `{"app_id":`,
		"JSONの後に値があった": `{"app_id":440} {}`, "数値がnullだった": `{"app_id":440,"max_reviews":null}`,
	} {
		t.Run(name+"ため拒否する", func(t *testing.T) {
			appID, options, err := parseInput([]byte(input))
			if err == nil || appID != 0 || options != (steam.CollectionOptions{}) {
				t.Fatalf("不正入力の結果 = %d, %+v, %v", appID, options, err)
			}
		})
	}
}

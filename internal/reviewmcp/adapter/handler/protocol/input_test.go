package protocol

import (
	"errors"
	"testing"

	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestParseInput(t *testing.T) {
	for _, tc := range []struct {
		name    string
		input   string
		appID   steamdomain.AppID
		options steamdomain.CollectionOptionsConfig
	}{
		{"省略時は少量の既定値となる", `{"app_id":440}`, 440, steamdomain.CollectionOptionsConfig{Language: "english", PageSize: 100, MaxReviews: 20, MaxPages: 3}},
		{"ゼロ値と空言語は既定値となる", `{"app_id":440,"language":"","page_size":0,"max_reviews":0,"max_pages":0}`, 440, steamdomain.CollectionOptionsConfig{Language: "english", PageSize: 100, MaxReviews: 20, MaxPages: 3}},
		{"上限と有効な言語を受け付ける", `{"app_id":4294967295,"language":"japanese","page_size":100,"max_reviews":100,"max_pages":5}`, 4294967295, steamdomain.CollectionOptionsConfig{Language: "japanese", PageSize: 100, MaxReviews: 100, MaxPages: 5}},
		{"最小の取得条件を受け付ける", `{"app_id":1,"language":"all","page_size":1,"max_reviews":1,"max_pages":1}`, 1, steamdomain.CollectionOptionsConfig{Language: "all", PageSize: 1, MaxReviews: 1, MaxPages: 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.options.ReviewType = steamdomain.ReviewTypeAll
			appID, options, err := ParseInput([]byte(tc.input))
			if err != nil || appID != tc.appID || options.Config() != tc.options {
				t.Fatalf("ParseInput = %d, %+v, %v", appID, options, err)
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
		"大文字の未定義項目があった": `{"APP_ID":440}`, "大文字の追加項目があった": `{"app_id":440,"MAX_PAGES":1}`,
	} {
		t.Run(name+"ため拒否する", func(t *testing.T) {
			appID, options, err := ParseInput([]byte(input))
			if !errors.Is(err, steamdomain.ErrInvalidArgument) || appID != 0 || options != (steamdomain.CollectionOptions{}) {
				t.Fatalf("不正入力の結果 = %d, %+v, %v", appID, options, err)
			}
		})
	}
}

func TestNormalizeToolInput(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input toolInput
		want  steamdomain.CollectionOptionsConfig
	}{
		{"省略とゼロ値へ MCP の既定値を適用する", toolInput{AppID: 440}, steamdomain.CollectionOptionsConfig{ReviewType: steamdomain.ReviewTypeAll, Language: "english", PageSize: 100, MaxReviews: 20, MaxPages: 3}},
		{"最小の条件とおすすめを変換する", toolInput{AppID: 1, ReviewType: "positive", Language: "all", PageSize: 1, MaxReviews: 1, MaxPages: 1}, steamdomain.CollectionOptionsConfig{ReviewType: steamdomain.ReviewTypePositive, Language: "all", PageSize: 1, MaxReviews: 1, MaxPages: 1}},
		{"最大の条件と非おすすめを変換する", toolInput{AppID: 4294967295, ReviewType: "negative", Language: "japanese", PageSize: 100, MaxReviews: 100, MaxPages: 5}, steamdomain.CollectionOptionsConfig{ReviewType: steamdomain.ReviewTypeNegative, Language: "japanese", PageSize: 100, MaxReviews: 100, MaxPages: 5}},
		{"明示した全評価と省略した条件を正規化する", toolInput{AppID: 440, ReviewType: "all"}, steamdomain.CollectionOptionsConfig{ReviewType: steamdomain.ReviewTypeAll, Language: "english", PageSize: 100, MaxReviews: 20, MaxPages: 3}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := tc.input
			appID, options, err := normalizeToolInput(input)
			if err != nil || appID != tc.input.AppID || options.Config() != tc.want {
				t.Fatalf("appID = %d, options = %+v, err = %v", appID, options, err)
			}

			if input != tc.input {
				t.Fatal("呼び出し側の入力を変更した")
			}
		})
	}

	for _, tc := range []struct {
		name  string
		input toolInput
	}{
		{"App ID がゼロ", toolInput{}},
		{"収集件数が負", toolInput{AppID: 440, MaxReviews: -1}},
		{"収集件数が MCP の上限を超える", toolInput{AppID: 440, MaxReviews: 101}},
		{"ページ数が負", toolInput{AppID: 440, MaxPages: -1}},
		{"ページ数が MCP の上限を超える", toolInput{AppID: 440, MaxPages: 6}},
		{"ページ件数が負", toolInput{AppID: 440, PageSize: -1}},
		{"ページ件数が共通の上限を超える", toolInput{AppID: 440, PageSize: 101}},
		{"言語が共通の構文を満たさない", toolInput{AppID: 440, Language: "日本語"}},
		{"評価種別が未定義", toolInput{AppID: 440, ReviewType: "mixed"}},
		{"評価種別が大文字", toolInput{AppID: 440, ReviewType: "Positive"}},
		{"評価種別に前後空白", toolInput{AppID: 440, ReviewType: " negative "}},
	} {
		t.Run(tc.name+"なら入力を変更せずゼロ値と原因を返す", func(t *testing.T) {
			input := tc.input
			appID, options, err := normalizeToolInput(input)
			if !errors.Is(err, steamdomain.ErrInvalidArgument) || appID != 0 || options != (steamdomain.CollectionOptions{}) {
				t.Fatalf("appID = %d, options = %+v, err = %v", appID, options, err)
			}

			if input != tc.input {
				t.Fatal("失敗時に呼び出し側の入力を変更した")
			}
		})
	}
}

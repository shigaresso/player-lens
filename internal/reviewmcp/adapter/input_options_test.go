package adapter

import (
	"errors"
	"testing"

	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

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

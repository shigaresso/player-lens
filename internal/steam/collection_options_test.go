package steam_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/shigaresso/player-lens/internal/steam"
)

func TestNormalizeCollectionOptions(t *testing.T) {
	cases := []struct {
		name    string
		input   steam.CollectionOptions
		want    steam.CollectionOptions
		invalid bool
	}{
		{"未指定は英語100件1000件100ページ", steam.CollectionOptions{}, steam.CollectionOptions{Language: "english", PageSize: 100, MaxReviews: 1000, MaxPages: 100}, false},
		{"最小の上限を保持する", steam.CollectionOptions{Language: "all", PageSize: 1, MaxReviews: 1, MaxPages: 1}, steam.CollectionOptions{Language: "all", PageSize: 1, MaxReviews: 1, MaxPages: 1}, false},
		{"最大の上限を保持する", steam.CollectionOptions{Language: "japanese", PageSize: 100, MaxReviews: 10000, MaxPages: 100}, steam.CollectionOptions{Language: "japanese", PageSize: 100, MaxReviews: 10000, MaxPages: 100}, false},
		{"ページ件数の負数を拒否する", steam.CollectionOptions{PageSize: -1}, steam.CollectionOptions{}, true},
		{"ページ件数101を拒否する", steam.CollectionOptions{PageSize: 101}, steam.CollectionOptions{}, true},
		{"収集件数の負数を拒否する", steam.CollectionOptions{MaxReviews: -1}, steam.CollectionOptions{}, true},
		{"収集件数10001を拒否する", steam.CollectionOptions{MaxReviews: 10001}, steam.CollectionOptions{}, true},
		{"ページ上限の負数を拒否する", steam.CollectionOptions{MaxPages: -1}, steam.CollectionOptions{}, true},
		{"ページ上限101を拒否する", steam.CollectionOptions{MaxPages: 101}, steam.CollectionOptions{}, true},
		{"不正な言語を共通の規則で拒否する", steam.CollectionOptions{Language: "Japanese"}, steam.CollectionOptions{}, true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.invalid {
				tt.want.ReviewType = steam.ReviewTypeAll
			}
			input := tt.input
			got, err := steam.NormalizeCollectionOptions(input)
			if !reflect.DeepEqual(got, tt.want) || errors.Is(err, steam.ErrInvalidArgument) != tt.invalid || (err != nil) != tt.invalid {
				t.Fatalf("options=%#v error=%v", got, err)
			}
			if input != tt.input {
				t.Fatal("入力が変更された")
			}
			if !tt.invalid {
				again, err := steam.NormalizeCollectionOptions(got)
				if err != nil || again != got {
					t.Fatalf("再適用=%#v error=%v", again, err)
				}
			}
		})
	}
	if steam.MaxCollectionReviews != 10000 || steam.MaxCollectionPages != 100 {
		t.Fatal("収集全体の上限が異なる")
	}
}

package domain_test

import (
	"errors"
	"reflect"
	"testing"

	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestNormalizeCollectionOptions(t *testing.T) {
	cases := []struct {
		name    string
		input   steamdomain.CollectionOptions
		want    steamdomain.CollectionOptions
		invalid bool
	}{
		{"未指定は英語100件1000件100ページ", steamdomain.CollectionOptions{}, steamdomain.CollectionOptions{Language: "english", PageSize: 100, MaxReviews: 1000, MaxPages: 100}, false},
		{"最小の上限を保持する", steamdomain.CollectionOptions{Language: "all", PageSize: 1, MaxReviews: 1, MaxPages: 1}, steamdomain.CollectionOptions{Language: "all", PageSize: 1, MaxReviews: 1, MaxPages: 1}, false},
		{"最大の上限を保持する", steamdomain.CollectionOptions{Language: "japanese", PageSize: 100, MaxReviews: 10000, MaxPages: 100}, steamdomain.CollectionOptions{Language: "japanese", PageSize: 100, MaxReviews: 10000, MaxPages: 100}, false},
		{"ページ件数の負数を拒否する", steamdomain.CollectionOptions{PageSize: -1}, steamdomain.CollectionOptions{}, true},
		{"ページ件数101を拒否する", steamdomain.CollectionOptions{PageSize: 101}, steamdomain.CollectionOptions{}, true},
		{"収集件数の負数を拒否する", steamdomain.CollectionOptions{MaxReviews: -1}, steamdomain.CollectionOptions{}, true},
		{"収集件数10001を拒否する", steamdomain.CollectionOptions{MaxReviews: 10001}, steamdomain.CollectionOptions{}, true},
		{"ページ上限の負数を拒否する", steamdomain.CollectionOptions{MaxPages: -1}, steamdomain.CollectionOptions{}, true},
		{"ページ上限101を拒否する", steamdomain.CollectionOptions{MaxPages: 101}, steamdomain.CollectionOptions{}, true},
		{"不正な言語を共通の規則で拒否する", steamdomain.CollectionOptions{Language: "Japanese"}, steamdomain.CollectionOptions{}, true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.invalid {
				tt.want.ReviewType = steamdomain.ReviewTypeAll
			}
			input := tt.input
			got, err := steamdomain.NormalizeCollectionOptions(input)
			if !reflect.DeepEqual(got, tt.want) || errors.Is(err, steamdomain.ErrInvalidArgument) != tt.invalid || (err != nil) != tt.invalid {
				t.Fatalf("options=%#v error=%v", got, err)
			}
			if input != tt.input {
				t.Fatal("入力が変更された")
			}
			if !tt.invalid {
				again, err := steamdomain.NormalizeCollectionOptions(got)
				if err != nil || again != got {
					t.Fatalf("再適用=%#v error=%v", again, err)
				}
			}
		})
	}
	if steamdomain.MaxCollectionReviews != 10000 || steamdomain.MaxCollectionPages != 100 {
		t.Fatal("収集全体の上限が異なる")
	}
}

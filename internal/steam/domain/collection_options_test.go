package domain_test

import (
	"errors"
	"reflect"
	"testing"

	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestNewCollectionOptions(t *testing.T) {
	cases := []struct {
		name    string
		input   steamdomain.CollectionOptionsConfig
		want    steamdomain.CollectionOptionsConfig
		invalid bool
	}{
		{"未指定は英語100件1000件100ページ", steamdomain.CollectionOptionsConfig{}, steamdomain.CollectionOptionsConfig{Language: "english", PageSize: 100, MaxReviews: 1000, MaxPages: 100}, false},
		{"最小の上限を保持する", steamdomain.CollectionOptionsConfig{Language: "all", PageSize: 1, MaxReviews: 1, MaxPages: 1}, steamdomain.CollectionOptionsConfig{Language: "all", PageSize: 1, MaxReviews: 1, MaxPages: 1}, false},
		{"最大の上限を保持する", steamdomain.CollectionOptionsConfig{Language: "japanese", PageSize: 100, MaxReviews: 10000, MaxPages: 100}, steamdomain.CollectionOptionsConfig{Language: "japanese", PageSize: 100, MaxReviews: 10000, MaxPages: 100}, false},
		{"ページ件数の負数を拒否する", steamdomain.CollectionOptionsConfig{PageSize: -1}, steamdomain.CollectionOptionsConfig{}, true},
		{"ページ件数101を拒否する", steamdomain.CollectionOptionsConfig{PageSize: 101}, steamdomain.CollectionOptionsConfig{}, true},
		{"収集件数の負数を拒否する", steamdomain.CollectionOptionsConfig{MaxReviews: -1}, steamdomain.CollectionOptionsConfig{}, true},
		{"収集件数10001を拒否する", steamdomain.CollectionOptionsConfig{MaxReviews: 10001}, steamdomain.CollectionOptionsConfig{}, true},
		{"ページ上限の負数を拒否する", steamdomain.CollectionOptionsConfig{MaxPages: -1}, steamdomain.CollectionOptionsConfig{}, true},
		{"ページ上限101を拒否する", steamdomain.CollectionOptionsConfig{MaxPages: 101}, steamdomain.CollectionOptionsConfig{}, true},
		{"不正な言語を共通の規則で拒否する", steamdomain.CollectionOptionsConfig{Language: "Japanese"}, steamdomain.CollectionOptionsConfig{}, true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.invalid {
				tt.want.ReviewType = steamdomain.ReviewTypeAll
			}

			input := tt.input
			got, err := steamdomain.NewCollectionOptions(input)
			if errors.Is(err, steamdomain.ErrInvalidArgument) != tt.invalid || (err != nil) != tt.invalid {
				t.Fatalf("options=%#v error=%v", got, err)
			}

			if input != tt.input {
				t.Fatal("入力が変更された")
			}

			if tt.invalid {
				if got != (steamdomain.CollectionOptions{}) {
					t.Fatalf("拒否時の値=%#v", got)
				}

				return
			}

			if !reflect.DeepEqual(got.Config(), tt.want) {
				t.Fatalf("options=%#v; want=%#v", got.Config(), tt.want)
			}

			again, err := steamdomain.NewCollectionOptions(got.Config())
			if err != nil || again != got {
				t.Fatalf("再適用=%#v error=%v", again, err)
			}
		})
	}

	if steamdomain.MaxCollectionReviews != 10000 || steamdomain.MaxCollectionPages != 100 {
		t.Fatal("収集全体の上限が異なる")
	}
}

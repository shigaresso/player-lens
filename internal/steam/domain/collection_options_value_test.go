package domain_test

import (
	"testing"

	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestCollectionOptionsZero(t *testing.T) {
	defaults := steamdomain.CollectionOptionsConfig{ReviewType: steamdomain.ReviewTypeAll, Language: "english", PageSize: 100, MaxReviews: 1000, MaxPages: 100}
	for _, tc := range []struct {
		name     string
		generate bool
		input    steamdomain.CollectionOptionsConfig
	}{
		{"ゼロ値を既定の有効な条件として使える", false, steamdomain.CollectionOptionsConfig{}},
		{"未指定の設定から同じゼロ値を生成する", true, steamdomain.CollectionOptionsConfig{}},
		{"明示した既定条件も同じゼロ値を生成する", true, defaults},
	} {
		t.Run(tc.name, func(t *testing.T) {
			options := steamdomain.DefaultCollectionOptions()
			if tc.generate {
				var err error
				options, err = steamdomain.NewCollectionOptions(tc.input)
				if err != nil {
					t.Fatal(err)
				}
			}

			if options != (steamdomain.DefaultCollectionOptions()) || options.Config() != defaults {
				t.Fatalf("options=%+v config=%+v", options, options.Config())
			}
		})
	}
}

func TestCollectionOptionsConfigCopy(t *testing.T) {
	custom := steamdomain.CollectionOptionsConfig{ReviewType: steamdomain.ReviewTypeNegative, Language: "japanese", PageSize: 1, MaxReviews: 2, MaxPages: 3}
	defaults := steamdomain.CollectionOptionsConfig{ReviewType: steamdomain.ReviewTypeAll, Language: "english", PageSize: 100, MaxReviews: 1000, MaxPages: 100}
	for _, tc := range []struct {
		name   string
		input  steamdomain.CollectionOptionsConfig
		want   steamdomain.CollectionOptionsConfig
		source bool
	}{
		{"生成元の変更は生成値へ影響しない", custom, custom, true},
		{"取得したコピーの変更は生成値へ影響しない", custom, custom, false},
		{"ゼロ値から取得したコピーも独立している", steamdomain.CollectionOptionsConfig{}, defaults, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := tc.input
			options, err := steamdomain.NewCollectionOptions(input)
			if err != nil {
				t.Fatal(err)
			}

			if tc.source {
				input.Language = "invalid language"
				input.MaxReviews = -1
			} else {
				snapshot := options.Config()
				snapshot.Language = "invalid language"
				snapshot.MaxReviews = -1
			}

			if got := options.Config(); got != tc.want {
				t.Fatalf("config=%+v; want=%+v", got, tc.want)
			}
		})
	}
}

func TestDefaultCollectionOptions(t *testing.T) {
	options := steamdomain.DefaultCollectionOptions()
	want := steamdomain.CollectionOptionsConfig{ReviewType: steamdomain.ReviewTypeAll, Language: "english", PageSize: 100, MaxReviews: 1000, MaxPages: 100}
	if options.Config() != want {
		t.Fatalf("既定条件を生成する: config=%+v", options.Config())
	}
}

package domain_test

import (
	"errors"
	"reflect"
	"testing"

	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestNewCollectionState(t *testing.T) {
	for _, tc := range []struct {
		name    string
		appID   steamdomain.AppID
		options steamdomain.CollectionOptionsConfig
		want    steamdomain.CollectionOptionsConfig
		size    int
	}{
		{"既定値を適用し先頭の100件から開始する", 440, steamdomain.CollectionOptionsConfig{}, steamdomain.CollectionOptionsConfig{ReviewType: steamdomain.ReviewTypeAll, Language: "english", PageSize: 100, MaxReviews: 1000, MaxPages: 100}, 100},
		{"指定条件を保持して開始する", 440, steamdomain.CollectionOptionsConfig{ReviewType: steamdomain.ReviewTypeNegative, Language: "japanese", PageSize: 2, MaxReviews: 3, MaxPages: 2}, steamdomain.CollectionOptionsConfig{ReviewType: steamdomain.ReviewTypeNegative, Language: "japanese", PageSize: 2, MaxReviews: 3, MaxPages: 2}, 2},
		{"件数上限がページ件数より小さければ要求件数を減らす", 440, steamdomain.CollectionOptionsConfig{MaxReviews: 1}, steamdomain.CollectionOptionsConfig{ReviewType: steamdomain.ReviewTypeAll, Language: "english", PageSize: 100, MaxReviews: 1, MaxPages: 100}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state, err := steamdomain.NewCollectionState(tc.appID, collectionOptions(t, tc.options))
			if err != nil {
				t.Fatal(err)
			}
			got := state.Result()
			want := steamdomain.Collection{AppID: tc.appID, Options: tc.want, Reviews: []steamdomain.Review{}}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("result=%#v; want=%#v", got, want)
			}
			next, ok := state.NextPage()
			if !ok || next != (steamdomain.PageOptions{ReviewType: tc.want.ReviewType, Language: tc.want.Language, Cursor: steamdomain.FirstPageCursor, PageSize: tc.size}) {
				t.Fatalf("next=%#v available=%v", next, ok)
			}
		})
	}
	for _, tc := range []struct {
		name    string
		appID   steamdomain.AppID
		options steamdomain.CollectionOptionsConfig
	}{
		{"App ID が0なら状態を作らない", 0, steamdomain.CollectionOptionsConfig{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state, err := steamdomain.NewCollectionState(tc.appID, collectionOptions(t, tc.options))
			if state != nil || !errors.Is(err, steamdomain.ErrInvalidArgument) {
				t.Fatalf("state=%v err=%v", state, err)
			}
		})
	}
}

func collectionOptions(t *testing.T, config steamdomain.CollectionOptionsConfig) steamdomain.CollectionOptions {
	t.Helper()
	options, err := steamdomain.NewCollectionOptions(config)
	if err != nil {
		t.Fatal(err)
	}
	return options
}

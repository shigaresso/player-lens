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
		options steamdomain.CollectionOptions
		want    steamdomain.CollectionOptions
		size    int
	}{
		{"既定値を適用し先頭の100件から開始する", 440, steamdomain.CollectionOptions{}, steamdomain.CollectionOptions{ReviewType: steamdomain.ReviewTypeAll, Language: "english", PageSize: 100, MaxReviews: 1000, MaxPages: 100}, 100},
		{"指定条件を保持して開始する", 440, steamdomain.CollectionOptions{ReviewType: steamdomain.ReviewTypeNegative, Language: "japanese", PageSize: 2, MaxReviews: 3, MaxPages: 2}, steamdomain.CollectionOptions{ReviewType: steamdomain.ReviewTypeNegative, Language: "japanese", PageSize: 2, MaxReviews: 3, MaxPages: 2}, 2},
		{"件数上限がページ件数より小さければ要求件数を減らす", 440, steamdomain.CollectionOptions{MaxReviews: 1}, steamdomain.CollectionOptions{ReviewType: steamdomain.ReviewTypeAll, Language: "english", PageSize: 100, MaxReviews: 1, MaxPages: 100}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state, err := steamdomain.NewCollectionState(tc.appID, tc.options)
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
		options steamdomain.CollectionOptions
	}{
		{"App ID が0なら状態を作らない", 0, steamdomain.CollectionOptions{}},
		{"不正な言語なら状態を作らない", 440, steamdomain.CollectionOptions{Language: "Japanese"}},
		{"件数上限の超過なら状態を作らない", 440, steamdomain.CollectionOptions{MaxReviews: 10001}},
		{"ページ上限の超過なら状態を作らない", 440, steamdomain.CollectionOptions{MaxPages: 101}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state, err := steamdomain.NewCollectionState(tc.appID, tc.options)
			if state != nil || !errors.Is(err, steamdomain.ErrInvalidArgument) {
				t.Fatalf("state=%v err=%v", state, err)
			}
		})
	}
}

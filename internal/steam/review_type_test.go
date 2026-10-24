package steam_test

import (
	"errors"
	"testing"

	"github.com/shigaresso/player-lens/internal/steam"
)

func TestReviewTypeNormalization(t *testing.T) {
	for _, tt := range []struct {
		name    string
		input   steam.ReviewType
		want    steam.ReviewType
		invalid bool
	}{
		{"ゼロ値は全評価", 0, 0, false},
		{"おすすめを保持する", 1, 1, false},
		{"非おすすめを保持する", 2, 2, false},
		{"負の数値を拒否する", -1, 0, true},
		{"未知の3を拒否する", 3, 0, true},
		{"int32最大値を拒否する", 2147483647, 0, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			pageInput := steam.PageOptions{ReviewType: tt.input}
			page, pageErr := steam.NormalizePageOptions(pageInput)
			collectionInput := steam.CollectionOptions{ReviewType: tt.input}
			collection, collectionErr := steam.NormalizeCollectionOptions(collectionInput)
			if errors.Is(pageErr, steam.ErrInvalidArgument) != tt.invalid || errors.Is(collectionErr, steam.ErrInvalidArgument) != tt.invalid {
				t.Fatalf("page error=%v collection error=%v", pageErr, collectionErr)
			}
			if tt.invalid {
				if page != (steam.PageOptions{}) || collection != (steam.CollectionOptions{}) {
					t.Fatalf("拒否時に結果を返した: page=%+v collection=%+v", page, collection)
				}
				return
			}
			if pageErr != nil || collectionErr != nil || page.ReviewType != tt.want || collection.ReviewType != tt.want {
				t.Fatalf("page=%+v collection=%+v errors=%v/%v", page, collection, pageErr, collectionErr)
			}
			againPage, err := steam.NormalizePageOptions(page)
			if err != nil || againPage != page {
				t.Fatalf("page 再適用=%+v err=%v", againPage, err)
			}
			againCollection, err := steam.NormalizeCollectionOptions(collection)
			if err != nil || againCollection != collection {
				t.Fatalf("collection 再適用=%+v err=%v", againCollection, err)
			}
			if pageInput.ReviewType != tt.input || collectionInput.ReviewType != tt.input {
				t.Fatal("入力を変更した")
			}
		})
	}
}

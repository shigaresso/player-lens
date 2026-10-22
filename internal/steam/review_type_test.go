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
		{"省略は全評価", "", steam.ReviewTypeAll, false},
		{"全評価を保持する", steam.ReviewTypeAll, steam.ReviewTypeAll, false},
		{"おすすめを保持する", steam.ReviewTypePositive, steam.ReviewTypePositive, false},
		{"非おすすめを保持する", steam.ReviewTypeNegative, steam.ReviewTypeNegative, false},
		{"未知値を拒否する", "mixed", "", true},
		{"大文字を拒否する", "Positive", "", true},
		{"前後空白を拒否する", " negative ", "", true},
		{"数値表記を拒否する", "1", "", true},
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

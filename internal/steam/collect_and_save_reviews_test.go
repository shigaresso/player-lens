package steam_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/shigaresso/player-lens/internal/steam"
)

type collectionSaverFunc func(context.Context, steam.Collection) error

func (f collectionSaverFunc) SaveCollection(ctx context.Context, result steam.Collection) error {
	return f(ctx, result)
}

func TestCollectAndSaveReviews(t *testing.T) {
	var events []string
	var saved steam.Collection
	fetcher := pageFetcherFunc(func(context.Context, uint32, steam.PageOptions) (steam.ReviewPage, error) {
		events = append(events, "fetch")
		return steam.ReviewPage{Reviews: collectionReviews("1"), NextCursor: "next"}, nil
	})
	saver := collectionSaverFunc(func(_ context.Context, result steam.Collection) error {
		events = append(events, "save")
		saved = result
		return nil
	})
	result, err := steam.CollectAndSaveReviews(t.Context(), fetcher, saver, 440, steam.CollectionOptions{MaxReviews: 1})
	if err != nil || !reflect.DeepEqual(saved, result) || result.StopReason != steam.StopReviewLimit || !reflect.DeepEqual(events, []string{"fetch", "save"}) {
		t.Fatalf("result=%+v saved=%+v events=%v err=%v", result, saved, events, err)
	}
}

func TestCollectAndSaveReviewsInvalidInput(t *testing.T) {
	for _, test := range []struct {
		name     string
		appID    uint32
		options  steam.CollectionOptions
		nilSaver bool
	}{
		{"AppIDが0ならI/Oしない", 0, steam.CollectionOptions{}, false},
		{"条件が不正ならI/Oしない", 440, steam.CollectionOptions{MaxReviews: -1}, false},
		{"保存先がnilならI/Oしない", 440, steam.CollectionOptions{}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			fetcher := pageFetcherFunc(func(context.Context, uint32, steam.PageOptions) (steam.ReviewPage, error) {
				t.Fatal("入力不正で取得した")
				return steam.ReviewPage{}, nil
			})
			var saver steam.CollectionSaver = collectionSaverFunc(func(context.Context, steam.Collection) error {
				t.Fatal("入力不正で保存した")
				return nil
			})
			if test.nilSaver {
				saver = nil
			}
			result, err := steam.CollectAndSaveReviews(t.Context(), fetcher, saver, test.appID, test.options)
			if !errors.Is(err, steam.ErrInvalidArgument) || !reflect.DeepEqual(result, steam.Collection{}) {
				t.Fatalf("result=%+v err=%v", result, err)
			}
		})
	}
}

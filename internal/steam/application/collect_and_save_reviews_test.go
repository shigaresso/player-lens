package application_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	steamapplication "github.com/shigaresso/player-lens/internal/steam/application"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

type collectionSaverFunc func(context.Context, steamdomain.Collection) error

func (f collectionSaverFunc) SaveCollection(ctx context.Context, result steamdomain.Collection) error {
	return f(ctx, result)
}

func TestCollectAndSaveReviews(t *testing.T) {
	var events []string
	var saved steamdomain.Collection
	fetcher := pageFetcherFunc(func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
		events = append(events, "fetch")
		return steamdomain.ReviewPage{Reviews: collectionReviews(1), NextCursor: "next"}, nil
	})
	saver := collectionSaverFunc(func(_ context.Context, result steamdomain.Collection) error {
		events = append(events, "save")
		saved = result
		return nil
	})
	result, err := steamapplication.CollectAndSaveReviews(t.Context(), fetcher, saver, 440, steamdomain.CollectionOptions{MaxReviews: 1})
	if err != nil || !reflect.DeepEqual(saved, result) || result.StopReason != steamdomain.StopReviewLimit || !reflect.DeepEqual(events, []string{"fetch", "save"}) {
		t.Fatalf("result=%+v saved=%+v events=%v err=%v", result, saved, events, err)
	}
}

func TestCollectAndSaveReviewsInvalidInput(t *testing.T) {
	for _, test := range []struct {
		name     string
		appID    steamdomain.AppID
		options  steamdomain.CollectionOptions
		nilSaver bool
	}{
		{"AppIDが0ならI/Oしない", 0, steamdomain.CollectionOptions{}, false},
		{"条件が不正ならI/Oしない", 440, steamdomain.CollectionOptions{MaxReviews: -1}, false},
		{"保存先がnilならI/Oしない", 440, steamdomain.CollectionOptions{}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			fetcher := pageFetcherFunc(func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
				t.Fatal("入力不正で取得した")
				return steamdomain.ReviewPage{}, nil
			})
			var saver steamapplication.CollectionSaver = collectionSaverFunc(func(context.Context, steamdomain.Collection) error {
				t.Fatal("入力不正で保存した")
				return nil
			})
			if test.nilSaver {
				saver = nil
			}
			result, err := steamapplication.CollectAndSaveReviews(t.Context(), fetcher, saver, test.appID, test.options)
			if !errors.Is(err, steamdomain.ErrInvalidArgument) || !reflect.DeepEqual(result, steamdomain.Collection{}) {
				t.Fatalf("result=%+v err=%v", result, err)
			}
		})
	}
}

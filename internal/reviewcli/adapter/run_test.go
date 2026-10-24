package adapter_test

import (
	"context"
	"errors"
	"flag"
	"io"
	"testing"

	reviewcli "github.com/shigaresso/player-lens/internal/reviewcli/adapter"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

type fetcherFunc func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error)

func (f fetcherFunc) FetchReviewsPage(ctx context.Context, id steamdomain.AppID, options steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
	return f(ctx, id, options)
}

type saverFunc func(context.Context, steamdomain.Collection) error

func (f saverFunc) SaveCollection(ctx context.Context, result steamdomain.Collection) error {
	return f(ctx, result)
}

func TestRunArguments(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
		want error
	}{
		{"helpは取得と保存をしない", []string{"--help"}, flag.ErrHelp},
		{"AppID必須", nil, reviewcli.ErrUsage},
		{"AppIDが0", []string{"--app-id=0"}, reviewcli.ErrUsage},
		{"AppIDが負数", []string{"--app-id=-1"}, reviewcli.ErrUsage},
		{"AppIDがuint32を超える", []string{"--app-id=4294967296"}, reviewcli.ErrUsage},
		{"AppIDが数値ではない", []string{"--app-id=abc"}, reviewcli.ErrUsage},
		{"値がない", []string{"--app-id"}, reviewcli.ErrUsage},
		{"未知のflag", []string{"--unknown"}, reviewcli.ErrUsage},
		{"位置引数は拒否", []string{"--app-id=440", "extra"}, reviewcli.ErrUsage},
		{"言語コード不正", []string{"--app-id=440", "--language=bad language"}, reviewcli.ErrUsage},
		{"件数が上限超過", []string{"--app-id=440", "--max-reviews=10001"}, reviewcli.ErrUsage},
		{"要求件数が上限超過", []string{"--app-id=440", "--page-size=101"}, reviewcli.ErrUsage},
		{"ページ上限が負数", []string{"--app-id=440", "--max-pages=-1"}, reviewcli.ErrUsage},
	} {
		t.Run(test.name, func(t *testing.T) {
			fetcher := fetcherFunc(func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
				t.Fatal("入力不正で通信した")
				return steamdomain.ReviewPage{}, nil
			})
			saver := saverFunc(func(context.Context, steamdomain.Collection) error {
				t.Fatal("入力不正で保存した")
				return nil
			})
			if err := reviewcli.Run(t.Context(), test.args, fetcher, saver); !errors.Is(err, test.want) {
				t.Fatalf("err=%v want=%v", err, test.want)
			}
		})
	}
}

func TestRunWithInjectedSaver(t *testing.T) {
	for _, test := range []struct {
		name     string
		args     []string
		id       steamdomain.AppID
		options  steamdomain.CollectionOptions
		fetchErr error
		saveErr  error
	}{
		{"既定値で保存する", []string{"--app-id=440"}, 440, steamdomain.CollectionOptions{Language: "english", PageSize: 100, MaxReviews: 1000, MaxPages: 100}, nil, nil},
		{"指定条件で別の保存先を使う", []string{"--app-id=4294967295", "--language=japanese", "--page-size=2", "--max-reviews=3", "--max-pages=4"}, 4294967295, steamdomain.CollectionOptions{Language: "japanese", PageSize: 2, MaxReviews: 3, MaxPages: 4}, nil, nil},
		{"途中失敗も保存して原因を返す", []string{"--app-id=440"}, 440, steamdomain.CollectionOptions{Language: "english", PageSize: 100, MaxReviews: 1000, MaxPages: 100}, io.ErrUnexpectedEOF, io.ErrClosedPipe},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.options.ReviewType = steamdomain.ReviewTypeAll
			calls, saves := 0, 0
			fetcher := fetcherFunc(func(_ context.Context, id steamdomain.AppID, options steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
				calls++
				if id != test.id || options.Language != test.options.Language || options.PageSize != test.options.PageSize || options.Cursor != steamdomain.FirstPageCursor {
					t.Fatalf("取得入力: id=%d options=%+v", id, options)
				}
				return steamdomain.ReviewPage{}, test.fetchErr
			})
			saver := saverFunc(func(_ context.Context, result steamdomain.Collection) error {
				saves++
				if result.AppID != test.id || result.Options != test.options {
					t.Fatalf("保存入力: %+v", result)
				}
				return test.saveErr
			})
			err := reviewcli.Run(t.Context(), test.args, fetcher, saver)
			if calls != 1 || saves != 1 || (test.fetchErr == nil && test.saveErr == nil && err != nil) {
				t.Fatalf("calls=%d saves=%d err=%v", calls, saves, err)
			}
			if test.fetchErr != nil && !errors.Is(err, test.fetchErr) {
				t.Fatalf("収集の原因: %v", err)
			}
			if test.saveErr != nil && !errors.Is(err, test.saveErr) {
				t.Fatalf("保存の原因: %v", err)
			}
		})
	}
}

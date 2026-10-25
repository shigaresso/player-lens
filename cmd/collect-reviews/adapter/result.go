package adapter

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"

	reviewcli "github.com/shigaresso/player-lens/internal/reviewcli/adapter"
	steamapplication "github.com/shigaresso/player-lens/internal/steam/application"
)

// Run は CLI の結果を表示し、成功・help は0、入力不正は2、実行失敗は1を返す。
// 診断の書き込みに失敗した場合も1とする。取得と保存の依存は呼び出し側が用意する。
func Run(ctx context.Context, args []string, fetcher steamapplication.PageFetcher, saver steamapplication.CollectionSaver, stderr io.Writer) int {
	err := reviewcli.Run(ctx, args, fetcher, saver)
	if err == nil {
		return 0
	}

	if errors.Is(err, flag.ErrHelp) {
		if _, err := fmt.Fprint(stderr, reviewcli.Usage); err != nil {
			return 1
		}

		return 0
	}

	code := 1
	if errors.Is(err, reviewcli.ErrUsage) {
		code = 2
	}

	return Report(stderr, err, code)
}

// Report は診断を stderr へ書き、指定した終了コードを返す。書き込み失敗は1とする。
func Report(stderr io.Writer, err error, code int) int {
	if _, err := fmt.Fprintln(stderr, err); err != nil {
		return 1
	}

	return code
}

// Package adapter は収集 CLI の入力解析と共通操作への接続を担当する。
package adapter

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"

	steamapplication "github.com/shigaresso/player-lens/internal/steam/application"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

// ErrUsage は通信前に CLI の入力が拒否されたことを表す。
var ErrUsage = errors.New("invalid review CLI usage")

// Usage は標準エラーへ表示する CLI の使用方法を表す。
const Usage = `使用方法: collect-reviews --app-id <作品ID> [オプション]
  --app-id       正の Steam App ID（必須、uint32）
  --language     レビューの言語（既定 english）
  --review-type  all・positive・negative（既定 all、空も all）
  --page-size    1リクエストの要求件数（1〜100、0は100）
  --max-reviews  収集する件数の上限（1〜10000、0は1000）
  --max-pages    ページ取得回数の上限（1〜100、0は100）
  --help         この使用方法を表示する
`

// Run は CLI の入力を解析し、注入された取得・保存先で収集操作を実行する。
// 出力形式を決めず、help は flag.ErrHelp、不正入力は ErrUsage を返す。
// 実行失敗は収集・保存の原因を保持する。表示と終了コードは入口側が担当する。
func Run(ctx context.Context, args []string, fetcher steamapplication.PageFetcher, saver steamapplication.CollectionSaver) error {
	flags := flag.NewFlagSet("collect-reviews", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	appID := flags.String("app-id", "", "Steam App ID")
	reviewType := flags.String("review-type", "", "レビューの評価種別")
	config := steamdomain.CollectionOptionsConfig{}
	flags.StringVar(&config.Language, "language", "", "レビューの言語")
	flags.IntVar(&config.PageSize, "page-size", 0, "要求件数")
	flags.IntVar(&config.MaxReviews, "max-reviews", 0, "レビュー上限")
	flags.IntVar(&config.MaxPages, "max-pages", 0, "ページ上限")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return flag.ErrHelp
		}
		return fmt.Errorf("parse review collection arguments: %w", errors.Join(ErrUsage, err))
	}
	id, err := strconv.ParseUint(*appID, 10, 32)
	if err != nil || id == 0 || flags.NArg() != 0 {
		return fmt.Errorf("positive uint32 --app-id and no positional arguments are required: %w", ErrUsage)
	}
	config.ReviewType, err = steamdomain.ParseReviewType(*reviewType)
	if err != nil {
		return fmt.Errorf("prepare review collection arguments: %w", errors.Join(ErrUsage, err))
	}
	options, err := steamdomain.NewCollectionOptions(config)
	if err != nil {
		return fmt.Errorf("prepare review collection arguments: %w", errors.Join(ErrUsage, err))
	}
	_, err = steamapplication.CollectAndSaveReviews(ctx, fetcher, saver, steamdomain.AppID(id), options)
	return err
}

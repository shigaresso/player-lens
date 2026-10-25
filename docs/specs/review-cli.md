# レビュー収集 CLI と保存先の契約

CLI の入力処理は reviewcli/adapter、JSON と writer は jsonoutput/adapter に置く。各領域の境界を維持する。 配置は [ADR-0013](../adr/0013-domain-folder-layers.md) に従う。

## 範囲と責務

[Issue #23](https://github.com/shigaresso/player-lens/issues/23) でローカル用 CLI を導入した。`CollectReviews` は取得のみ、`CollectAndSaveReviews` は収集と結果保存の順序・失敗の扱いを担当する。`CollectionSaver.SaveCollection(context.Context, Collection) error` を操作側で定義し、保存先を注入する。JSON・将来の DB 保存の選択は起動部分で行い、収集処理と CLI の入力処理を変更しない。[ADR-0006](../adr/0006-collection-output-boundary.md) と [操作一覧](../use-cases.md) を参照する。

`reviewcli.Run` は引数の解析と共通操作の呼び出しを担当する。`cmd/collect-reviews` は HTTP クライアント・JSON Saver・標準出力・シグナルを組み立て、診断と終了コードを担当する。再試行・ページの走査・重複排除を入口へ重複させない。

## CLI の入力と終了

| 引数 | 契約 |
| --- | --- |
| --app-id | 必須。十進数の正の uint32 |
| --review-type | 既定 all。all・positive・negative。空も all |
| --language | 既定 english。言語コードの構文は収集の共通検査に従う |
| --page-size | 1〜100。0・省略は100 |
| --max-reviews | 1〜10000。0・省略は1000 |
| --max-pages | 1〜100。0・省略は100 |
| --help / -h | 使用方法を標準エラーへ出し、通信・保存しない |

未知の flag、余分な位置引数、必須値の欠落・範囲外は通信・保存前に拒否する。標準 flag の解析を使い、flag と値を通常の引数として渡す。flag の値を未検証の `CollectionOptionsConfig` に保持し、`NewCollectionOptions` で有効な条件を生成してから操作へ渡す。生成エラーは利用方法のエラーとして処理し、既定条件での実行に置き換えない。

正常な収集と保存、help は終了コード0、不正入力は2、収集・保存・診断出力の失敗は1とする。stdout は JSON のみ、help と診断は stderr に出す。保存先を別実装へ差し替えても `reviewcli.Run` と収集・保存操作は同じものを使う。

## 保存の順序と失敗

有効な依存・生成済みの CollectionOptions と非 nil の context を渡す。保存先が nil、または収集の入力が不正ならゼロ値と原因を保持したエラーを返し、I/O しない。収集操作が結果を初期化した後は、正常結果・途中失敗・キャンセル・最初の取得失敗も含めて結果を保存先へ1回渡す。成功レビューがない失敗も、条件と `failed` を保持する。

保存の成功は収集失敗を取り消さない。保存の失敗は収集の終了理由を書き換えない。両方が失敗した場合は `errors.Join` と操作の文脈で両方の原因を保持し、`errors.Is` / `errors.As` で識別できる。保存や収集の自動再試行は行わない。

収集の context は既存の期限・キャンセルを尊重する。保存用 context は親の値を保持し、親のキャンセル・期限から切り離して最大5秒の期限を設定する。収集キャンセル後の部分結果を保存するための明示的な方針とする。操作が成功・失敗とも保存用 context を解放する。保存実装は期限を尊重し、goroutine を放置して処理終了を装わない。

## JSON 変換と writer

`jsonoutput.MarshalCollection` はメモリ内の Collection を変換する純粋な処理とし、通信・保存しない。出力形式を領域の型や Steam の応答形式と分離する。

| JSON 項目 | 内容 |
| --- | --- |
| schema_version | 1。JSON の形式を識別する |
| app_id | 作品 ID |
| options | review_type・language・page_size・max_reviews・max_pages |
| reviews | id・text・language・recommended・created_at・updated_at を取得順で保持する配列 |
| pages_fetched | 空ページを含む成功した取得回数 |
| stop_reason | exhausted・review_limit・page_limit・failed |

Issue #33 の `options.review_type` は schema_version=1 への追加項目とする。旧出力で欠落している場合は従来の全評価を意味し、既存項目の意味を変えない。Collection.Options は表示・保存用の CollectionOptionsConfig のコピーとする。空の Collection の言語・件数などは未指定のまま出力し、自動的に既定条件へ変換しない。通常の収集結果は正規化済みの条件を出力し、純粋な変換で取得件数等の正規化を重複させない。評価種別は文字列への変換時に未知の数値を拒否する。

Issue #43 では領域の RecommendationID（uint64）を10進の JSON 文字列として出力し、0と最大値を精度損失なく保持する。元の先頭ゼロは保持しない。AppID・時刻は名前付き数値型から既存の数値 JSON に変換し、元データの時刻は UNIX 秒とする。空のレビューは `[]` とし、本文を JSON の文字列として扱う。本文の指示を実行しない。入力を変更しない。

`jsonoutput.NewSaver` は利用可能な writer を受け取り、nil を拒否する。`SaveCollection` は JSON と改行を1回書き込み、書き込み失敗・短い書き込みを原因付きのエラーにする。writer の同期・Close は呼び出し側が担当し、Saver は Close しない。

書き込み前に context を検査する。通常の `io.Writer` の実行中の Write は context で中断できず、任意の writer の厳密な5秒終了は保証しない。書き込みのエラーでは一部が出力済みの可能性がある。JSON 出力は永続化・原子的保存の保証を持たず、取得時刻・本文版・分析対象集合を管理する保存機能は別 Issue で設計する。

## Docker と検証

Compose の collect サービスで読み取り専用のソースをビルドし、CLI バイナリを exec する。サーバーの PORT の契約を維持し、CLI は本番用イメージへ含めない。ビルドと収集コマンドを分け、JSON にビルドログを混ぜない。使い方は [README](../../README.md#レビュー収集-cli) に記載する。

Dockerfile の共通 `GOFLAGS=-buildvcs=false` を使い、ローカル CLI とテスト内の子ビルドもコンテナの Git 所有権・権限・有無に依存する VCS 情報の埋め込みを省略する。個別のビルド引数へ重複してフラグを追加しない。適用範囲と制約は [ADR-0001](../adr/0001-runtime-and-deployment.md#git-情報に依存しない-go-実行issue-23) に従う。ソースの対象 SHA は GitHub CI と作業記録で別に確認する。Git の取得失敗をアプリケーションの失敗へ混入させず、実 CLI プロセスの検査を維持する。

通常テストは取得・保存のテスト実装と httptest を使い、Steam へ接続しない。実 Client と JSON Saver の連携、別実装のメモリ Saver を確認する。integration タグのプロセステストはバイナリをビルドし、help・不正入力の終了コードと出力を外部通信なしで確認する。実 DB・MCP・永続保存と同等の保証とは扱わない。検証結果は [作業記録](../worklogs/23-review-collection-cli.md) に残す。

Issue #43 の評価種別は内部では ReviewType を使い、HTTP adapter で EUserReviewsReviewType に変換する。CLI 入力の文字列は ParseReviewType で解析し、JSON 出力は FormatReviewType で all/positive/negative に戻す。未知の数値は出力時も拒否する。ゼロ値は全評価なので、空の Collection の options.review_type も all とする。

# Steam レビューの上限付き収集

## 範囲と依存

[Issue #21](https://github.com/shigaresso/player-lens/issues/21) の `steam.CollectReviews` の契約を定める。[1ページ取得 adapter](steam-review-page.md) を利用し、カーソルの走査・重複排除・終了条件を担当する。操作一覧は [ユースケース](../use-cases.md)、判断は [ADR-0005](../adr/0005-bounded-review-collection.md) を参照する。

`PageFetcher` は利用側の1メソッドの interface とし、既存の `Client` を渡せる。依存は有効な `ReviewPage` を返す1ページ adapter の契約に従い、context を尊重する。HTTP クライアント・接続プールの組み立てと終了は呼び出し側、応答 Body の解放は adapter が担当する。収集操作内で HTTP クライアントを生成しない。

## 入力と正規化

`CollectReviews(ctx, fetcher, appID, options)` は正の App ID、利用可能な取得依存と非 nil の context を受ける。App ID が0、取得依存が nil、または取得条件が不正なら通信せず、`Collection{}` と `ErrInvalidArgument` を返す。interface に利用不能な値を渡さない責務は呼び出し側にある。

`NormalizeCollectionOptions` は通信なしで既定値を適用し、入力を変更せずコピーを返す。不正時はゼロ値と `ErrInvalidArgument` を返す。正規化済みの値への再適用は値を変えない。通常の struct なので、書き換え後の有効性を型で保証しない。

| 項目 | 既定値 | 有効範囲 |
| --- | --- | --- |
| ReviewType | all | all・positive・negative（空は all） |
| Language | english | `NormalizePageOptions` と同じ API コード構文 |
| PageSize | 100 | 1〜`MaxPageSize`（100） |
| MaxReviews | 1000 | 1〜`MaxCollectionReviews`（10000） |
| MaxPages | 100 | 1〜`MaxCollectionPages`（100） |

0は既定値の指定とし、負数と上限超過を拒否する。1ページ100件は Steam の仕様、収集全体の件数・ページ数・時間はアプリケーション側の制限であり、Steam の QPS を表さない。対応言語は Steam 側に依存する。

## ページの走査と結果

- `FirstPageCursor` から始め、取得した `NextCursor` を加工せず次の入力へ渡す。
- 1ページの要求件数は `min(PageSize, MaxReviews - 重複除外後の件数)` とする。
- `RecommendationID` の数値が同じレビューは1回の収集内で一度だけ返し、最初に取得した本文・時刻などの値と取得順を保持する。短いページや重複だけのページでも、終了条件に達していなければ続ける。
- `Collection` は AppID、正規化した元の Options、Reviews、PagesFetched、StopReason を保持する。PagesFetched は空ページを含む成功ページ数で、失敗したページは含めない。成功したレビューが0件でも Reviews は空配列とする。

終了の判定順序は空ページ、レビュー件数上限、カーソル循環、ページ上限とする。循環判定は取得に使った全カーソルとの比較で行う。

| 状況 | 終了理由 | エラー・結果 |
| --- | --- | --- |
| 空ページを取得した | StopExhausted | 成功。対象条件の走査終了 |
| 重複除外後の件数が上限に達した | StopReviewLimit | 成功。全件取得とは扱わない |
| 成功ページ数が上限に達した | StopPageLimit | 成功。全件取得とは扱わない |
| 次のカーソルが取得済みのカーソルに戻った | StopFailed | 取得済み結果と ErrCursorCycle |
| ページ取得・context が失敗した | StopFailed | 取得済み結果と原因を保持したエラー |

空ページを観測したことと、更新中の Steam レビューを同一時点のスナップショットとして取得できたことは区別する。評価種別は共通のページ正規化で検査し、正規化した条件を Collection と各ページ要求へ保持する。部分失敗でも条件を失わせない。言語・ソート・off-topic の条件は1ページ adapter の仕様に従う。部分結果に永続化・再開の保証は付けない。ページ取得 adapter は失敗時にゼロ値を返し、収集操作がそれ以前の成功結果を保持する。

## 期限と失敗

収集全体を最大2分に制限し、呼び出し側の短い期限と context 値・キャンセルを伝播する。各ページ取得前にも context の終了を検査し、終了済みなら次の取得を呼ばない。派生 context は成功・失敗とも収集操作が解放し、呼び出し側の context は終了させない。1ページの10秒上限は adapter が維持する。

収集操作からの再試行は0回とする。HTTP 429、通信・読み取り・期限・キャンセルなどの失敗で直ちに終了し、`%w` で原因を保持する。`errors.Is` / `errors.As` で1ページ adapter の複数原因とステータスを確認できる。取得依存の内部に再試行や期限を無視する処理を混入させない。

## 検証と未導入項目

通常テストは制御した `PageFetcher` の応答を使う。実 `Client` との接続も `httptest` のローカルサーバーで確認し、Steam へ接続しない。最大件数、重複・順序、カーソルと残件数、途中失敗・原因・終了優先順位、期限・キャンセル・解放を検証する。実 API は少量の手動確認として区別し、テスト・CI に組み込まない。

DB、取得時刻・本文版の永続化、ジョブ・再開、再試行・待機・並列取得、Web の入口、バックエンドの分析は未導入とする。MCP の入口は Issue #25 で追加し、[MCP 仕様](review-mcp.md) のより小さい上限で同じ操作を使う。検証履歴は [作業記録](../worklogs/21-collect-reviews.md) に残す。

Issue #23 で [収集 CLI](review-cli.md) と保存先を注入する別の操作を導入した。CollectReviews の取得のみの契約は維持し、部分結果を保存する順序・複合失敗・保存期限はその操作に分離する。

Issue #43 では AppID とレビューの ID・時刻を名前付き数値型で受け渡す。0001 と 1 のように同じ数値を表す ID は同一とし、0も通常の ID として重複排除する。

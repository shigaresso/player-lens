# Steam レビューの上限付き収集

操作と利用側の interface は steam/application、型・正規化と収集状態は steam/domain、具体的な HTTP は steam/adapter に置く。 配置は [ADR-0013](../adr/0013-domain-folder-layers.md) に従う。

## 範囲と依存

[Issue #21](https://github.com/shigaresso/player-lens/issues/21) の `steamapplication.CollectReviews` の契約を定める。[1ページ取得 adapter](steam-review-page.md) を利用し、カーソルの走査・重複排除・終了条件を担当する。操作一覧は [ユースケース](../use-cases.md)、判断は [ADR-0005](../adr/0005-bounded-review-collection.md) を参照する。

`PageFetcher` は利用側の1メソッドの interface とし、既存の `Client` を渡せる。依存は有効な `ReviewPage` を返す1ページ adapter の契約に従い、context を尊重する。HTTP クライアント・接続プールの組み立てと終了は呼び出し側、応答 Body の解放は adapter が担当する。収集操作内で HTTP クライアントを生成しない。

## 入力と正規化

`CollectReviews(ctx, fetcher, appID, options)` は正の App ID、利用可能な取得依存と非 nil の context を受ける。options は有効な `CollectionOptions` を渡す。App ID が0なら通信せず、`Collection{}` と `ErrInvalidArgument` を返す。取得依存の存在は生成・組み立て側の契約とし、利用不能な interface を渡さない責務は呼び出し側にある。Issue #105 で利用側の nil 再検査を削除した。現在の静的解析は対象型の生成・更新境界を検査し、取得依存の非 nil は生成・組み立て側と呼び出し元の契約とする。

状態生成で App ID を確認する。nil の取得依存は正常な入力エラーとして扱わず、生成済みの依存を渡す事前条件への違反となる。生成・更新側と静的検査の責務は [ADR-0019](../adr/0019-construction-contracts.md) に記載する。

`CollectionOptionsConfig` は公開フィールドを持つ未検証の設定、`CollectionOptions` は非公開フィールドで有効な条件を保持する値とする。`NewCollectionOptions(config)` が通信なしで既定値と検査・正規化を適用し、入力を変更せずコピーを保持する。不正時はゼロ値と `ErrInvalidArgument` を保持するエラーを返す。呼び出し側はエラーを処理してから収集操作へ渡す。公開の NormalizeCollectionOptions は生成 API へ移行し、互換ラッパーは残さない。

内部の `CollectionOptions{}` は以下の有効な既定条件を表す。Issue #107 では外側の直接構築を禁止し、既定条件は `DefaultCollectionOptions()` で生成する。明示した既定条件の生成値も同じゼロ値に揃える。`Config()` は設定のコピーを返し、生成元やコピーの変更は生成済みの値に影響しない。コピーからの再生成は同じ値になる。利用側は条件を再検査せず、ゼロ値からの取得も固定の既定値を返す。判断は [ADR-0015](../adr/0015-collection-options-boundary.md) に記録する。

| 項目 | 既定値 | 有効範囲 |
| --- | --- | --- |
| ReviewType | ReviewTypeAll（0） | ReviewTypeAll・ReviewTypePositive・ReviewTypeNegative（CLI・MCP の文字列の空は all） |
| Language | english | `NormalizePageOptions` と同じ API コード構文 |
| PageSize | 100 | 1〜`MaxPageSize`（100） |
| MaxReviews | 1000 | 1〜`MaxCollectionReviews`（10000） |
| MaxPages | 100 | 1〜`MaxCollectionPages`（100） |

0は既定値の指定とし、負数と上限超過を拒否する。1ページ100件は Steam の仕様、収集全体の件数・ページ数・時間はアプリケーション側の制限であり、Steam の QPS を表さない。対応言語は Steam 側に依存する。

## ページの走査と結果

- `FirstPageCursor` から始め、取得した `NextCursor` を加工せず次の入力へ渡す。
- 1ページの要求件数は `min(PageSize, MaxReviews - 重複除外後の件数)` とする。
- `RecommendationID` の数値が同じレビューは1回の収集内で一度だけ返し、最初に取得した本文・時刻などの値と取得順を保持する。短いページや重複だけのページでも、終了条件に達していなければ続ける。
- `Collection` は AppID、元の条件のコピー（Options: CollectionOptionsConfig）、Reviews、PagesFetched、StopReason を保持する。PagesFetched は空ページを含む成功ページ数で、失敗したページは含めない。成功したレビューが0件でも Reviews は空配列とする。

終了の判定順序は空ページ、レビュー件数上限、カーソル循環、ページ上限とする。循環判定は取得に使った全カーソルとの比較で行う。

| 状況 | 終了理由 | エラー・結果 |
| --- | --- | --- |
| 空ページを取得した | StopExhausted | 成功。対象条件の走査終了 |
| 重複除外後の件数が上限に達した | StopReviewLimit | 成功。全件取得とは扱わない |
| 成功ページ数が上限に達した | StopPageLimit | 成功。全件取得とは扱わない |
| 次のカーソルが取得済みのカーソルに戻った | StopFailed | 取得済み結果と ErrCursorCycle |
| ページ取得・context が失敗した | StopFailed | 取得済み結果と原因を保持したエラー |

空ページを観測したことと、更新中の Steam レビューを同一時点のスナップショットとして取得できたことは区別する。評価種別は共通のページ正規化で検査し、正規化した条件を Collection と各ページ要求へ保持する。部分失敗でも条件を失わせない。言語・ソート・off-topic の条件は1ページ adapter の仕様に従う。部分結果に永続化・再開の保証は付けない。ページ取得 adapter は失敗時にゼロ値を返し、収集操作がそれ以前の成功結果を保持する。

## 収集状態の境界（Issue #58）

steam/domain の CollectionState は結果・既読 ID・既読カーソルと次の取得条件を非公開で持つ。NewCollectionState が正の App ID を検査し、有効な CollectionOptions の設定コピーを保持する。状態のゼロ値は使用しない。条件の検査・正規化は NewCollectionOptions で完了しており、状態では繰り返さない。CollectReviews は生成済みの取得依存を受け取って状態を生成し、App ID の検査を重複させない。不正入力の戻り値と ErrInvalidArgument の原因識別は維持する。

NextPage は次の条件と続行可否、AddPage は成功ページの反映と循環のエラー、Fail は取得や context 失敗の結果への反映を担当する。Issue #86 で、AddPage の重複排除は同じ CollectionState の非公開メソッド appendUnseenReviews に分離した。既読 ID の確認・登録と配列への追加をまとめ、終了判定の位置と優先順を維持する。終了後は NextPage がゼロ値と false を返し、AddPage と Fail は結果を変更しない。終了後の AddPage は nil を返す。

Result はレビュー配列も含めてコピーし、返した値の変更を内部へ伝えない。空配列は nil にしない。状態は通信・context・時計・再試行を扱わず、取得・期限・キャンセルと失敗の文脈は application が担当する。循環の識別には steam/domain.ErrCursorCycle を使う。判断は [ADR-0014](../adr/0014-steam-collection-state.md) に記録する。

### 失敗結果の共通生成（Issue #84）

`NewFailedCollection(Collection)` は入力の App ID・収集条件・取得済みレビュー・成功ページ数を保持した失敗結果のコピーを返す。終了理由を StopFailed にし、レビュー配列も複製する。nil と空配列の区別を保ち、入力や返却後のコピーの変更が互いのレビューへ影響しない。値の検査・既定値の適用・context・原因の分類は担当しない。

CollectionState.Fail は進行中の結果をこの共通生成へ渡し、カーソル循環時も同じ Fail で確定する。終了済みの状態は変更しない。MCP の開始前キャンセルと受付拒否も同じ生成処理を使う。

空の失敗と部分結果の失敗は、渡す Collection の内容で表す。失敗にしたい結果を明示して渡し、入力不正で返す Collection のゼロ値をこの処理で失敗結果へ変えない。

## 期限と失敗

収集全体を最大2分に制限し、呼び出し側の短い期限と context 値・キャンセルを伝播する。各ページ取得前にも context の終了を検査し、終了済みなら次の取得を呼ばない。派生 context は成功・失敗とも収集操作が解放し、呼び出し側の context は終了させない。1ページの10秒上限は adapter が維持する。

収集操作からの再試行は0回とする。HTTP 429、通信・読み取り・期限・キャンセルなどの失敗で直ちに終了し、`%w` で原因を保持する。`errors.Is` / `errors.As` で1ページ adapter の複数原因とステータスを確認できる。取得依存の内部に再試行や期限を無視する処理を混入させない。

ページ取得の失敗には既存の1始まりのページ番号を付ける。Issue #90 で1ページ adapter のレビュー変換失敗に同じく1始まりのレビュー番号を付け、原因を引き継いだエラーからページとページ内の位置を追えるようにした。失敗したページは部分的に取り込まず、それ以前の成功ページを収集結果に保持する。

## 検証と未導入項目

通常テストは制御した `PageFetcher` の応答を使う。実 `Client` との接続も `httptest` のローカルサーバーで確認し、Steam へ接続しない。最大件数、重複・順序、カーソルと残件数、途中失敗・原因・終了優先順位、期限・キャンセル・解放を検証する。実 API は少量の手動確認として区別し、テスト・CI に組み込まない。

DB、取得時刻・本文版の永続化、ジョブ・再開、再試行・待機・並列取得、Web の入口、バックエンドの分析は未導入とする。MCP の入口は Issue #25 で追加し、[MCP 仕様](review-mcp.md) のより小さい上限で同じ操作を使う。検証履歴は [作業記録](../worklogs/21-collect-reviews.md) に残す。

Issue #23 で [収集 CLI](review-cli.md) と保存先を注入する別の操作を導入した。CollectReviews の取得のみの契約は維持し、部分結果を保存する順序・複合失敗・保存期限はその操作に分離する。

Issue #43 では AppID とレビューの ID・時刻を名前付き数値型で受け渡す。0001 と 1 のように同じ数値を表す ID は同一とし、0も通常の ID として重複排除する。

# ADR-0014：収集状態とページ反映のルールを domain にまとめる

## 状態と対象

採用。[Issue #58](https://github.com/shigaresso/player-lens/issues/58) の Steam 収集操作に適用する。ADR-0005 の上限・部分結果と ADR-0013 の各領域内のレイヤーを維持し、収集状態の管理を steam/domain に配置する。

条件の検査・正規化の担当は [ADR-0015](0015-collection-options-boundary.md) で部分的に置き換えた。状態は生成済みの CollectionOptions を受け取り、App ID の検査と設定のコピーの保持を担当する。以下の重複・終了・部分結果・資源の契約は維持する。

## 背景と比較

CollectReviews が取得の手順と result・seenIDs・seenCursors・cursor を個別に管理していた。重複と終了判断には、最初の値・順序、成功ページ数、終了の優先順位という関係がある。

- 個別の map や終了判定を引数で渡す関数へ分割する案は、状態の整合性を呼び出し側に残すため採用しない。
- 収集の I/O と context を domain へ移す案は、純粋な収集ルールと外部依存が混ざるため採用しない。
- CollectionState が状態と更新を所有し、application が取得・期限・原因の文脈を担当する構成を採用する。

## 決定

NewCollectionState は正の App ID と CollectionOptions を検査・正規化して生成する。条件検査を application と重ねず、非公開の結果と既読の ID・カーソル、次のカーソルを保持する。ゼロ値は使用しない。

NextPage は残件数までに制限した取得条件と続行可否を返す。AddPage は有効な取得結果を順番に受け取り、成功数と重複排除・終了・次の条件をまとめて更新する。空ページ、件数上限、カーソル循環、ページ上限の順序を維持する。循環したページも結果に含め、ErrCursorCycle を domain に置く。参照元は新しい定義へ更新し、互換エイリアスを残さない。

Issue #86 では AddPage の既読 ID の確認・登録とレビュー配列への追加を、同じ型の非公開メソッド appendUnseenReviews に分離する。AddPage はページ反映と終了判断の順序を担当し、重複排除の詳細は意味のある名前で参照する。状態の所有・公開 API・終了優先順は変えず、新しいパッケージや interface は追加しない。

Fail は外側の取得・context 失敗を収集結果へ反映し、成功済みのページと条件を保持する。原因のエラーや通信の文脈は application が保持する。終了済みなら NextPage はゼロ値と false、AddPage は変更せず nil、Fail は何も変更しない。

Result はレビュー配列を含むコピーを返す。外部からの結果変更で、内部のレビューと既読 ID の整合性を壊さない。空のレビュー配列を nil にしない。

CollectReviews は PageFetcher の確認、状態の生成、最大2分の context、取得・失敗・結果返却を担当する。既存のエラー原因・部分結果・再試行なし・終了時の context 解放を維持する。新たなユースケースや通信境界は増やさない。

## 失敗結果の共通生成（Issue #84）

NewFailedCollection は Collection の入力条件・取得済みレビュー・成功ページ数を保ち、StopFailed とレビュー配列のコピーを持つ失敗結果を生成する。CollectionState.Fail は進行中の結果をこの処理へ渡す。カーソル循環も同じ Fail で確定し、終了済み状態を変更しない契約を維持する。

MCP の開始前キャンセルもこの生成を使う。キャンセルを判定して早期 return する責務は application に置き、domain に context やエラー分類を持ち込まない。空の結果を作る処理で部分結果を置き換えず、同じ生成へそれぞれの結果を渡す。失敗結果のコピーにより失敗確定時にもレビュー配列の複製が発生する。

## 利点と制約

application から状態の個別更新を除き、domain のテストで収集の判断を通信・時計なしで確認できる。生成と各ページの反映は同じ収集から順に呼び出し、並行更新は扱わない。

Result ごとに保持レビュー数に応じたコピーを行う。application は正常終了か失敗時の結果返却に使い、各ページ取得のたびにコピーしない。入力エラーはゼロ値と ErrInvalidArgument を保持するエラーを維持し、App ID の拒否は状態生成の文脈で返す。

## 参照

- [収集仕様](../specs/collect-reviews.md)「入力と正規化」「ページの走査と結果」「期限と失敗」
- [ユースケース](../use-cases.md)、ADR-0005・0012・0013

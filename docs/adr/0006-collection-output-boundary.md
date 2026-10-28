# ADR-0006：収集結果の保存先を利用側の契約で切り替える

## 状態と対象

採用。[Issue #23](https://github.com/shigaresso/player-lens/issues/23) の収集 CLI・収集と保存の操作に適用する。[仕様](../specs/review-cli.md) に契約を記載する。DB 製品の採用・永続データ設計は決定しない。

配置の決定は [ADR-0013](0013-domain-folder-layers.md) で部分的に置き換えた。以下の収集・保存・失敗・資源の振る舞いは維持する。現在の配置は各領域内の domain・application・adapter とする。

## 背景と比較

ユーザーの方針として、JSON と将来の DB 保存を差し替え、収集や CLI の処理を共通で使う。形式の変換だけを差し替える設計では、保存先への I/O と失敗の扱いが入口へ混在する。

| 選択肢 | 判断 |
| --- | --- |
| CollectReviews に JSON や DB の処理を追加する | 取得と保存が結合するため採用しない |
| CLI に具体的な保存形式と部分失敗の判断を置く | MCP などの入口で重複するため採用しない |
| 利用側の小さな保存 interface と収集・保存操作を追加する | 保存先の選択を起動部分へ置けるため採用する |
| JSON の型をそのまま DB の入力型にする | 保存先が出力形式へ依存するため採用しない |

## 決定

`steam.CollectionSaver` に Collection と context を渡す。操作全体を担う `CollectAndSaveReviews` が収集・部分結果の保存・複合エラー・保存期限を管理する。委譲だけの互換ラッパーとはせず、取得のみの `CollectReviews` は既存の責務を維持する。

JSON の純粋変換と writer の adapter は `jsonoutput` に置く。CLI の入力は `reviewcli` に置き、main で具体的な取得と保存先を組み立てる。DB を導入する際は同じ保存契約を実装して組み立て箇所で選択する。DB の一意性・トランザクション・本文版・分析集合の契約は別途決める。

収集に失敗しても初期化済みの結果を一度保存する。保存で収集の失敗を取り消さず、両方の失敗を保持する。再試行は行わない。親がキャンセル済みでも部分結果の保存を試すため、保存は `context.WithoutCancel` で値を保持し、新しい5秒の期限を付けて解放する。収集側の短い期限・キャンセルは変更しない。

## JSON 出力データの構築（Issue #92）

MarshalCollection に変換・データ構築・JSON 化を残す案と比較し、同じ adapter の collection_document.go に非公開の出力型と newCollectionDocument を置く。評価種別とレビュー ID の変換、空配列と出力項目の対応をまとめ、MarshalCollection では構築と JSON 化の手順を読む。取得条件とレビューの初期化は項目名を明示し、フィールドの宣言順を照合せずに対応を確認できるようにする。

公開する DTO や汎用変換層、1項目ごとの helper は増やさない。構築に失敗した場合の文脈と原因、公開入口の nil の JSON、既存の schema_version=1 とすべての値・項目順・入力の非変更を維持する。MCP の応答と Saver は引き続き同じ MarshalCollection を呼ぶ。関数呼び出しは増えるが、JSON のエンコード詳細と出力形式への対応を分けて追える境界を採用する。

## 利点と制約

収集処理と CLI の入力処理が保存形式を知らず、JSON とメモリへの保存を同じ操作で検証できる。PostgreSQL の adapter・永続的な版管理は未導入とする。JSON の schema_version は出力形式の識別であり、レビュー本文の版とは異なる。

部分結果の保存に最大5秒の追加時間を割り当てる。保存依存が context を尊重する契約に依存する。通常の io.Writer の実行中の書き込みは中断できず、厳密な期限や原子的な出力を保証しない。失敗時の一部出力を再試行で重複させない。CLI はローカル用とし、サーバーの本番イメージや公開デモを変更しない。

CLI は閉じた stdout/stderr の pipe も出力エラーとして扱い、終了コード1を返す。起動時に SIGPIPE を無視し、書き込みエラーを通常の終了処理へ渡す。SIGINT・SIGTERM は収集のキャンセルに用いる。

## 参照

- [ADR-0005](0005-bounded-review-collection.md) と [データ・調査仕様](../specs/data-and-research.md)
- [Go context.WithoutCancel](https://pkg.go.dev/context#WithoutCancel)
- [Go io.Writer](https://pkg.go.dev/io#Writer)
- [Go os/signal の SIGPIPE](https://pkg.go.dev/os/signal#hdr-SIGPIPE)

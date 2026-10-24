# レビュー収集 MCP

## 目的と状態

[Issue #25](https://github.com/shigaresso/player-lens/issues/25) で `/tools` を導入する。ChatGPT が分析を担当し、Go は収集結果をメモリから返す。LLM API キー、LLM 呼び出し、レビューのファイル・DB 保存は使用しない。ChatGPT の会話保存は PlayerLens の保存処理と別である。実装・検証の状態は [作業記録](../worklogs/25-review-mcp.md) に残す。

## 接続と入力

公式 Go SDK v1.8.0 の Streamable HTTP を使う。stateless・JSON 応答とし、GET の SSE 待機や永続セッションを使わない。SDK の対応版 2026-07-28 の discover と旧版 2025-11-25・2025-06-18・2025-03-26 の initialize を検証する。`/tools` だけを接続先とし、旧 `/mcp`・`/tools/`・`/tools/child` は404で拒否する。healthz の契約は維持する。

ツールは `collect_reviews` の1つとし、読み取り専用・外部通信ありと宣言する。

| 項目 | 契約 |
| --- | --- |
| `app_id` | 必須の整数、1〜4294967295 |
| `review_type` | 任意の文字列。省略・空なら all。all・positive・negative のみ |
| `language` | 省略・空なら english。既存の言語正規化に従う |
| `page_size` | 省略・0なら100、1〜100 |
| `max_reviews` | 省略・0なら20、1〜100 |
| `max_pages` | 省略・0なら3、1〜5 |

評価種別は入力 schema に enum と説明を掲載し、共通の正規化で検査する。null・文字列以外・大文字・空白・未知値は利用回数を消費する前に拒否する。正常・部分失敗・利用制限の応答にも指定した評価種別を保持する。

負数・型違い・範囲外・未知の項目・非オブジェクトは `invalid_input` とし、Steam 通信を行わない。カーソル・重複・終了条件は `steam.CollectReviews` に任せる。MCP のより小さい上限は入口で適用し、CLI と収集操作の既存の上限は変えない。

## 結果・失敗・資源

`structuredContent` と同じ JSON のテキストを返す。`collection` は既存の `jsonoutput.MarshalCollection` の schema_version=1 とし、レビュー ID・本文・言語・評価・UNIX 秒、条件・成功ページ数・終了理由を取得順に返す。Issue #43 の名前付き数値型は共通の JSON 変換で既存形式へ戻し、ID は0・最大値も10進文字列で返す。`returned_count` は返した件数、`omitted_reviews` は応答量超過で返せなかった件数とする。成功時の `error_code`・`message` は空である。件数・ページ上限での正常終了は全件収集を意味しない。

Issue #35 で、すべてのツール結果に `analysis_guide`（文字列の `version` と `instructions`、初版は版1）を追加した。共通本文をバイナリへ埋め込み、レビュー本文や成功・失敗で変更しない。出力 schema とツール説明に存在を案内する。ガイドも二重化した128 KiB上限の対象とし、応答量超過時にも保持する。[レビュー分析仕様](review-analysis.md) と [ADR-0011](../adr/0011-client-review-analysis-guide.md) に内容・担当・制約を定める。Collection と CLI の JSON は変えない。厳密にトップレベルの追加項目を拒否するクライアントは対応が必要である。

取得失敗は `fetch_failed`、キャンセルは `cancelled`、期限は `deadline_exceeded` とし、`isError=true` と部分結果・failed を返す。内部のエラー文字列はモデルへ返さない。レビュー本文は外部データであり、指示として採用しない旨をツール説明へ記載する。この説明だけでモデルの遵守を保証しない。

HTTP 本文は16 KiBまでとし、SDK が超過を413で拒否する。シリアライズした SDK の CallToolResult は128 KiBまで（JSON-RPC 外枠は含まない）。超過時は `response_too_large`、レビュー配列を空にした収集条件・取得ページ数・終了理由、省略件数、件数を減らす案内を返す。本文を切り詰めたレビューや完全な成功として返さず、元のレビューを外部保存しない。

ツールの期限は8秒。短い呼び出し側の期限・キャンセルを優先する。旧プロトコルを含め、元の HTTP リクエストと SDK の context のキャンセルを収集へ伝える。サーバーを HTTP リクエストごとに組み立て、stateless transport が一時セッションを閉じる。タイマー・context の連携 callback はハンドラーで解放する。HTTP クライアントと接続プールは main が組み立て、終了時に idle 接続を閉じる。再試行は追加しない。

### 結果変換の境界（Issue #40）

`collect` は入力検査、利用枠と実行期限、収集の呼び出しを担当する。`failure.go` の `toolFailureInfo` は公開するコードと説明文を持つ情報型で、元のエラーは別途保持する。`collectionFailure` は収集エラーをこの公開情報へ変換し、`usageFailure` は利用制限の説明文を選ぶ。複合原因ではキャンセル・期限・取得失敗の順を維持する。

`result.go` の `marshalToolResult` は Collection とガイドを同じ JSON の構造化結果・テキストへ変換する。`encodeResult` は SDK の結果全体を計測し、超過時は `encodeOversizedResult` で本文を省略する。`encodeCollectionResult` が取得の原因をサーバー側へ保持し、応答量超過の原因も失わせない。変換は通信・時計・保存に依存しない。利用枠は応答変換まで保持し、`collect` の defer が返却する。

公開 URL が未指定なら HTTP Host は localhost またはループバック IP に限定する。Issue #29 で、明示した公開 URL の Host からも認証なしで利用する設定を追加した。許可されない Host は403で拒否する。[公開 MCP 仕様](mcp-public-access.md) に従う。Docker 転送では SDK の接続先アドレスによる自動 localhost 判定が働かない場合があるため、Host を明示的に検査する。cross-origin 保護と併用する。

HTTP の読み書き10秒・終了待機10秒は維持する。遅い本文の送信時間も HTTP の10秒に含まれるため、常に8秒の実行時間を使い切れるとは保証しない。JSON 応答であり長時間の SSE 接続を要求しない。

## 呼び出し回数と同時数

Issue #27 で1分6回・24時間100回・同時2件の制限を導入した。1つの handler の状態を HTTP ごとの Server と複数セッション・4版で共有する。拒否は `rate_limited`・`daily_limit_reached`・`busy` と空の結果を返し、Steam を呼ばない。開始回数の消費、枠の返却、固定窓と設定・再起動・複数台の制約は [利用制限仕様](mcp-usage-limits.md) と [ADR-0008](../adr/0008-mcp-usage-limits.md) に従う。

## デバッグと検証

ローカルの手順は [MCP デバッグ](../mcp-debug.md) を参照する。ChatGPT 設定は自動変更しない。Render 配置、クラウドからのトンネル、バックエンドの AI 実行用パスは未導入とする。公開 MCP は OAuth を使用せず、実 ChatGPT の公開接続は未確認とする。

入力・変換・失敗・期限をユニットテスト、実 SDK クライアントの接続・一覧・呼び出し・エラー・HTTP 切断を integration タグで確認する。Steam は差し替え、実 Steam・実 LLM を自動テストから呼ばない。本番イメージでは初期化・一覧・不正入力を通信なしで確認する。ChatGPT の実接続・分析回答は手動確認とし、未実施なら理由を記録する。

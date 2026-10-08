# ADR-0007：ローカルのレビュー収集 MCP

## 状態

採用。Issue #25 の方針を記録した。実装・検証の状態は作業記録を参照する。

## 背景と比較

API キーなしで ChatGPT アプリから収集結果を分析したい。独自の JSON-RPC 実装は互換性・資源管理の負担がある。CLI の JSON ファイルを経由すると保存・読み戻しが必要になる。公式 Go SDK と既存の収集操作を直接接続する方式を採用した。

## 決定

- [公式 Go SDK v1.8.0](https://github.com/modelcontextprotocol/go-sdk/releases/tag/v1.8.0) を固定した。Go 1.25 以上を要求するため既存の1.27.1で実行できる。go.mod・go.sum、Docker の依存ダウンロード、Compose Watch を管理する。
- ChatGPT の入力欄の `/mcp` コマンドと区別しやすくするため、URL のパスを `/tools` とする。ツール名とプロトコルは変更せず、旧 `/mcp` は公開しない。
- 同じ HTTP サーバーに `/tools` を追加し、stateless・JSON の Streamable HTTP を使う。永続セッション、サーバーからの sampling・elicitation・roots 要求を使わない。
- SDK の新旧プロトコルを利用し、旧版でも HTTP 切断を収集へ伝える。HTTP リクエストごとにサーバーとツールを組み立て、呼び出し context を共有可変状態へ格納しない。
- `steam.CollectReviews` を直接使い、純粋な `jsonoutput.MarshalCollection` を結果変換へ再利用する。保存ユースケースと CLI 出力は経由しない。
- 数値上限、8秒の実行期限、部分失敗と応答量超過は [MCP 仕様](../specs/review-mcp.md) に従う。HTTP の10秒の期限は変更しない。
- 分析を行うのは接続元の ChatGPT とする。将来バックエンドで LLM API を呼ぶ場合は別の HTTP 操作として決定し、この段階で API キーや空の分析層を追加しない。

## 利点と制約

公式クライアントでプロトコル連携を検証でき、収集の責務を共通化できる。短時間・少量のデバッグ用途であり、大規模分析・再開・永続的な根拠管理を実現しない。認証なしのローカル接続を対象にし、SDK の localhost 保護と Go の cross-origin 保護に加え、Docker 転送時にも localhost またはループバック IP の Host だけを受け付ける。Issue #27 と [ADR-0008](0008-mcp-usage-limits.md) で1プロセスの開始回数・同時実行数を制限した。公開サービスへの認証・利用期限・共有予算は別 Issue で導入する。

ChatGPT の実接続はアプリの設定・対応版に依存する。SDK 連携の成功だけで ChatGPT の分析結果を確認済みと扱わない。開発・検証の依存は Go のモジュールキャッシュへ置き、実行用イメージは引き続き CGO 不要のバイナリと証明書だけを含める。

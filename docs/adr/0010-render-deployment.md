# ADR-0010：検証済み GHCR digest を使う Render Free の配置準備

## 状態

採用。Issue #31 で配置準備を実装する。ユーザーは当初、サービス未作成のため配置準備まで進めると回答した。作業中にユーザー配置の公開 URL で基本通信を確認したが、こちらからの Git/registry への push・digest 配置・ChatGPT 接続は未実施である。結果は作業記録へ残す。

## 背景と比較

ADR-0001 の「一度ビルドして検証した同じイメージを digest で Render へ配置する」を具体化する。Git-backed の Docker 再ビルドは検証済み成果物と同一でなくなるため使わない。自動デプロイや deploy hook の秘密情報は今回必要としない。

GHCR は既存の GitHub と連携でき、private image を Render が読み取れる。Render での registry 認証には read:packages の資格情報を別途登録する。無料のポートフォリオとして Free・単一サービスを選び、idle 停止と起動待ちを許容する。永続 DB/ディスクや新しい LLM 依存は追加しない。

## 決定

main の成功した最新 push CI を確認する手動ワークフローから、linux/amd64 を一度ビルド・同じ image ID を検証して送信する。送信先 digest の config と image ID を照合して配置先を記録する。既存のテスト CI と成果物検査は省略しない。Render 設定は実 digest から生成し、初回はローカル専用で安全に起動してから公開 URL を明示する。詳細は [仕様](../specs/render-deployment.md) に記載する。

ADR-0002 の読み取り専用 CI に対し、今回追加する手動のイメージ公開ジョブだけ packages:write を許可する。既存 CI と Release 作成/公開の権限や手順は変更しない。main 以外の起動は拒否し、Render API の書き込み権限や自動的な package 公開化は追加しない。

## 利点と制約

ソース SHA・image ID・manifest digest を追跡できる。イメージが同じでも配置先の TLS・通信・資源は別なので実 URL の確認が必要である。Free の cold start は MCP クライアントの期限内に終わる保証がない。メモリ内の予算は停止で消え、利用者別予算や請求上限を保証しない。GitHub Actions の無料利用枠と registry の条件も確認が必要となる。

初回 Render のサービス名・URL・registry credential は外部で準備する。今回のテスト成功を registry 送信・Render 配置・ChatGPT の成功と扱わない。

## 公式資料

- [Render の prebuilt image](https://render.com/docs/deploying-an-image)
- [Blueprint 仕様](https://render.com/docs/blueprint-spec)
- [Free の制約](https://render.com/docs/free)
- [GHCR](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
- [Docker push](https://docs.docker.com/reference/cli/docker/image/push/)
- [manifest の確認](https://docs.docker.com/reference/cli/docker/buildx/imagetools/inspect/)

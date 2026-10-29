# ADR-0019：生成・更新の境界を静的検査で支える

## 状態と対象

採用。Issue #105 で、Limits・HTTPConfig・CollectionState の生成・更新と、NewHandler・CollectReviews の依存の契約に適用する。導入・実行結果は作業記録へ残し、方針の採用だけで検査済みとは扱わない。ADR-0008・0009・0014 の利用側の存在確認に関する記述を部分置換する。

## 背景と比較

生成側で有効な値を作った後、利用側でも依存の nil や設定のゼロ値を拒否していた。有効な依存・設定を渡す責務を生成・組み立て側へ集め、利用側はその契約を信頼する。これは任意の Go 呼び出しを安全にする変更ではなく、このリポジトリで守る API の事前条件である。

- 存在確認を各利用箇所へ置く案は、保証の責務が分散するため採用しない。
- 文書・レビューだけで生成経路を管理する案に対し、型情報を使った検査で違反を早く拒否する構成を採用する。
- 全 struct・標準型のゼロ値を禁止する案は、入力用の値・有効な既定値・検索結果などと区別できないため採用しない。

## 決定

型の宣言に `//contractcheck:controlled` を付ける。対象は非公開フィールドを持つ struct とし、同名の別型や import の別名と混同せず、型別名にも同じ規則を適用する。対象の宣言ファイル内で生成と状態更新を行い、それ以外のファイルでは生成済みの値をコピー・受け渡しして利用する。外部テストを含めて解析し、テストファイル全体の除外はしない。

独自検査は直接の複合リテラル、未初期化宣言、new、埋め込み先の省略フィールド・配列・make による暗黙の未初期化値、定義ファイル外からのフィールド更新・内部へのアドレス取得・ポインター経由の値の置き換えを拒否する。更新メソッドは有効な状態から有効な状態へ遷移させ、CollectionState.AddPage の終了判断や部分結果の契約を維持する。内部参照をそのまま外へ返す記述も検査し、コピーの独立性は型のテストで確認する。

生成成功なら有効な値・存在する依存を返し、失敗を呼び出し元で処理する。NewHandler と CollectReviews の取得依存は、生成成功した非 nil の値を渡す呼び出し側の契約とする。一般的な nil の流れを解析して非 nil を証明する専用ツールは使用しない。Issue #107 で利用側が必須引数を再検査する記述を禁止する検査を追加する。Limits の失敗値は NewLimits の生成結果を伝播し、入力解析側に直接の Limits{} を残さない。解析失敗はゼロの LimitsConfig を返す既存契約を維持し、NewLimits の結果とともに元の解析エラーを返す。解析失敗の原因を検証エラーで置き換えない。

既存 errcheck のエラー破棄の検査を維持する。当初導入した nilnil と NilAway は、ユーザーの追加指示によりいったん外す。実行・ビルド・専用注釈・専用 fixture と不要な依存を削除し、生成・更新境界の自作検査を維持する。既存 golangci-lint の配布バイナリ自体は変更せず、nilnil を有効な検査から外す。

GOFLAGS で integration を指定し、無効な -tags 互換フラグに依存しない。contractcheck は tools の独立モジュールに置き、x/tools とその依存を go.mod/go.sum に固定する。検証用イメージだけに配置し、本番の依存・イメージへ含めない。

NewHandler は生成済みの fetcher・Limits・HTTPConfig、CollectReviews は有効な PageFetcher を受け取り、存在を再検査しない。未生成の依存・設定を直接渡す行為は呼び出し側の契約違反となり、従来の入力エラー返却を保証しない。外部入力、AppID、応答欠落、利用枠、終了状態、資源解放の判断は残す。ゼロ値が有効な CollectionOptions と ValidatedPortNumber の契約は維持する。

## 検証と限界

型と生成・更新位置の検査、生成・状態更新の契約テストを組み合わせる。独自検査もツールの実行経路も、正常例・違反例と非破壊性をテストする。実アプリケーションの隔離コピーへ Limits・HTTPConfig・CollectionState の直接生成を挿入し、対象の宣言や接続が失われた場合も検証が失敗することを確認する。pre-push と go-lint CI は同じ読み取り専用コマンドを実行する。

配置の検査だけで任意の業務不変条件を証明しない。条件付きの取得・error・bool の扱いとコピーの深さはテスト・レビューでも確認する。nil の受け渡し全般、nil と成功の同時返却の専用解析は未導入とする。Issue #107 の利用側の再検査を禁止する構文検査は、呼び出し側が必ず有効な依存を渡すことの証明とは区別する。reflection・unsafe・cgo や外部コードによる任意の書き換え、全 nil panic の排除、競合の不存在まで証明したとは扱わない。

## 参照

- [開発手順](../development.md)、ADR-0001・0002・0003
- [利用制限](../specs/mcp-usage-limits.md)、[公開 MCP](../specs/mcp-public-access.md)、[収集仕様](../specs/collect-reviews.md)
- ADR-0014・0015・0018


## 全体への適用と利用側の再検査の禁止（Issue #107）

本番の cmd・internal の自作型に controlled・dependency・data の分類を必須とし、data には対象外の理由を記載する。入力 DTO・公開の API 数値型・出力・ログ・エラーのデータは未検証入力や値の受け渡しという契約を明記し、生成済みの有効性を保証する型とは区別する。テスト専用の宣言と検査ツール自身の型は本番型の分類から区別するが、テストから保証対象を直接構築・更新する記述は検査する。

保証対象は初回3型に CollectionOptions・ValidatedPortNumber・Client・Saver・Limiter・window・healthHandler・collectReviewsHandler・localAccess・publicAccess を加える。ゼロ値が有効でも外側の直接構築は許可せず、CollectionOptions の既定条件は定義ファイルの DefaultCollectionOptions、ポートは NewValidatedPortNumber の生成結果を使う。型内部のゼロ値と公開動作の意味は変えない。

必須引数の nil との比較、switch の nil 分岐、別名・クロージャ経由の同じ確認、保証対象の Validate・Valid・IsValid・IsZero・IsNil メソッドによる再検査を拒否する。error 引数は成功・失敗の結果として nil を許可し、ctx.Err・利用枠・終了・応答の状態判断は維持する。任意引数は optional に引数名と理由を明示する。生成関数の依存検査は constructor で指定し、同じ定義ファイルの保証対象を返す関数だけを認める。

Protect と CollectAndSaveReviews の存在確認を削除する。Protect から NewHandler まで生成エラーがなくなる経路の error 戻り値を削除し、設定不正の拒否は NewHTTPConfig に維持する。通常の外部入力の検査を禁止するものではなく、AppID・PageOptions・ReviewType の現行の未検証入力型の条件は入力・正規化・状態生成の境界で確認する。任意の helper や reflection・unsafe を含むコードの意味や、全 nil panic を証明するものではない。

# ADR-0018：検証済みポート番号と待ち受けアドレスの構築を分ける

## 状態と対象

採用。[Issue #99](https://github.com/shigaresso/player-lens/issues/99) に対応する。cmd/server の PORT の解析と待ち受けアドレスの構築を対象とし、設定の公開動作は維持する。

## 背景と比較

ListenAddress は入力の検査と待ち受けアドレスの構築をまとめて担当し、戻り値の string では検査済みのポート番号という契約を表せなかった。関数名だけを ParseListenAddress に変える案と、ポート番号の値型へ分ける案を比較し、生成と利用の責務を分ける後者を採用する。

非公開フィールドだけでは Go のゼロ値の生成を禁止できない。ゼロ値を未初期化として利用境界で拒否する案と、CollectionOptions と同様に有効な既定値とする案を比較し、既存の空入力の8080を維持して後者を採用する。

## 決定

- cmd/server/adapter の port_number.go に ValidatedPortNumber と NewValidatedPortNumber(string) を置く。外部設定の解析であり、純粋な処理でも業務上の domain は追加しない。
- 非公開の uint16 フィールドにより、外側から任意の内部値を設定できないようにする。生成後の変更 API は設けない。生成時だけ入力の ASCII 数字と1〜65535を検査する。
- 内部の0は既定の8080を表す。空入力と明示した8080も同じゼロ値に揃え、同じポート番号を値の比較でも同一にする。
- String は10進数のポート番号だけを返す。内部の0から8080への変換は値の表現であり、入力の解析や範囲検査を繰り返さない。
- run は生成エラーを処理してから、net.JoinHostPort の空 Host とポート番号を使って全インターフェースのアドレスを組み立てる。待ち受け可能かは net.Listen が判断する。旧 ListenAddress の互換ラッパーは残さない。

## 利点と制約

生成側が入力を検査し、利用側は有効なポート番号として扱える。待ち受け Host の指定と入力検査を別々に読める。通信、再試行、listener や接続の所有は変更しない。

ゼロ値も有効なので、生成エラー時の戻り値だけで成否を判断できず、必ず error を処理する。Validated は有効な値であることを表し、生成関数を実際に通った履歴を意味しない。同じパッケージ内の不正な直接構築や unsafe に対する保証は持たず、非公開フィールドの直接構築は生成関数に集約する。

## 参照

- [HTTP サーバー仕様](../specs/http-server.md) の「設定と終了の契約」「検証との対応」
- [ADR-0013](0013-domain-folder-layers.md) の設定解析の配置、[ADR-0015](0015-collection-options-boundary.md) のゼロ値と有効な既定値の考え方
- 実装と検証は [Issue #99 の記録](../worklogs/99-validated-port-number.md) に残す。


Issue #107 で ValidatedPortNumber を生成・更新境界の検査へ追加した。外側では有効なゼロ表現も直接作らず NewValidatedPortNumber の生成結果を使う。型内部のゼロ値が8080を表す意味は維持する。適用と限界は ADR-0019 に従う。

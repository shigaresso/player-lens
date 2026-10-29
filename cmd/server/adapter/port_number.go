package adapter

import (
	"fmt"
	"strconv"
)

const defaultPortNumber uint16 = 8080

// ValidatedPortNumber は1〜65535の有効なサーバーポート番号を保持する。
// ゼロ値は既定の8080を表す。生成後は変更せず、文字列表現には String を使う。
// ポート番号の有効性を表し、そのポートで待ち受け可能であることは保証しない。
//
//contractcheck:controlled
type ValidatedPortNumber struct {
	number uint16
}

// NewValidatedPortNumber は PORT の入力を検査し、有効なポート番号を生成する。
// 空は8080とし、ASCII 数字の1〜65535を受け付ける。通信や環境変数の取得は行わない。
// 不正ならゼロ値とエラーを返す。ゼロ値も有効なので、呼び出し側は必ずエラーを処理する。
func NewValidatedPortNumber(raw string) (ValidatedPortNumber, error) {
	if raw == "" {
		return ValidatedPortNumber{}, nil
	}

	number, err := strconv.ParseUint(raw, 10, 16)
	if err != nil || number == 0 {
		return ValidatedPortNumber{}, fmt.Errorf("PORT は1〜65535の数字で指定する: %q", raw)
	}

	if number == uint64(defaultPortNumber) {
		return ValidatedPortNumber{}, nil
	}

	return ValidatedPortNumber{number: uint16(number)}, nil
}

// String は先頭のゼロを除いた10進数のポート番号を返す。Host や区切りのコロンは含めない。
// ゼロ値からも8080を返し、入力の検査や正規化を繰り返さない。
func (port ValidatedPortNumber) String() string {
	number := port.number
	if number == 0 {
		number = defaultPortNumber
	}

	return strconv.FormatUint(uint64(number), 10)
}

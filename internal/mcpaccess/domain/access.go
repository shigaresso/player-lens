package domain

import (
	"errors"
	"fmt"
)

// ErrInvalidConfig は MCP の公開 URL または保護の組み立てが不正であることを表す。
var ErrInvalidConfig = errors.New("invalid MCP access configuration")

// Access は MCP の許可 Host の判定と公開設定の有無を提供する契約である。
// New で生成した値を渡す。ゼロ値の nil は未生成であり、ローカル用とは扱わない。
// 利用者の認証や外部通信は行わない。
type Access interface {
	// AllowsHost は Host が生成時の許可条件に一致するかを返す。
	AllowsHost(host string) bool
	// IsPublic は公開 URL を指定した設定かを返す。
	IsPublic() bool
}

// New は HTTPS の /tools の公開 URL を検査して許可条件を生成する。
// 空ならローカル用、指定時は公開用の非公開の値型を返す。
// 正常な結果に nil ポインターは含まない。不正設定の診断へ入力値を含めない。
// 外部通信やバックグラウンド処理は開始しない。
func New(publicURL string) (Access, error) {
	if publicURL == "" {
		return localAccess{}, nil
	}

	u, authority, err := parseHTTPSURL(publicURL)
	if err != nil || u.Path != "/tools" || u.RawPath != "" {
		return nil, fmt.Errorf("MCP_PUBLIC_URL は HTTPS の /tools を指定する: %w", ErrInvalidConfig)
	}

	return publicAccess{authority: authority}, nil
}

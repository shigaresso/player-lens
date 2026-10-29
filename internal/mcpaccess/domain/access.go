package domain

import "errors"

// ErrInvalidConfig は MCP の公開 URL または HTTP 保護の組み立てが不正であることを表す。
var ErrInvalidConfig = errors.New("invalid MCP access configuration")

// Access は MCP の許可 Host を判定する契約である。
// NewLocal または NewPublic で生成した値を渡す。
// 利用者の認証・表示・SDK 設定・外部通信は担当しない。
//
//contractcheck:dependency
type Access interface {
	// AllowsHost は Host が生成時の許可条件に一致するかを返す。
	AllowsHost(host string) bool
}

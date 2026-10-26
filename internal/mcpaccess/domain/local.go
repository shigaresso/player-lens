package domain

import (
	"net"
	"strings"
)

// localAccess は追加の設定を持たないローカル用の許可条件である。
type localAccess struct{}

// IsPublic はローカル用の設定であることを返す。
func (localAccess) IsPublic() bool { return false }

// HostRejectionMessage はローカル用の固定の Host 拒否理由を返す。
func (localAccess) HostRejectionMessage() string {
	return "ローカルの Host のみ利用できます"
}

// AllowsHost は従来のローカル判定で localhost・ループバックを許可する。
// 公開 authority の正規化は適用せず、DNS 照会や通信は行わない。
func (localAccess) AllowsHost(host string) bool {
	if name, _, err := net.SplitHostPort(host); err == nil {
		host = name
	}

	if strings.EqualFold(host, "localhost") {
		return true
	}

	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}

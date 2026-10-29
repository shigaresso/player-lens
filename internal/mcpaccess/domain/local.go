package domain

import (
	"net"
	"strings"
)

// localAccess は追加の設定を持たないローカル用の許可条件である。
//
//contractcheck:controlled
type localAccess struct{}

// NewLocal は従来のローカル Host 判定を行う許可条件を生成する。
// 外部通信やバックグラウンド処理は開始しない。
func NewLocal() Access { return localAccess{} }

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

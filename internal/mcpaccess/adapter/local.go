package adapter

import (
	"net"
	"net/http"
	"strings"
)

// ProtectLocal は localhost またはループバック IP の Host と cross-origin を next の手前で検査する。
// 不許可のリクエストは403で拒否し、next を呼ばない。Host の検査を先に行う。
// next は呼び出し側が用意した有効な非 nil のハンドラーを渡す。
func ProtectLocal(next http.Handler) http.Handler {
	protected := http.NewCrossOriginProtection().Handler(next)
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if !localHost(request.Host) {
			http.Error(w, "ローカルの Host のみ利用できます", http.StatusForbidden)
			return
		}

		protected.ServeHTTP(w, request)
	})
}

func localHost(host string) bool {
	if name, _, err := net.SplitHostPort(host); err == nil {
		host = name
	}

	if strings.EqualFold(host, "localhost") {
		return true
	}

	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}

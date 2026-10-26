package domain

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// ErrInvalidConfig は MCP の公開 URL または保護の組み立てが不正であることを表す。
var ErrInvalidConfig = errors.New("invalid MCP access configuration")

// Access は認証なしの MCP で許可する Host の条件を保持する。
// New で構築し、設定変更時は作り直す。利用者の認証は行わない。
type Access struct {
	authority string
	localOnly bool
}

// New は HTTPS の /tools の公開 URL を検査する。
// 空なら初期化済みのローカル用 Access を返す。不正設定の診断へ入力値を含めない。
// 外部通信やバックグラウンド処理は開始しない。
func New(publicURL string) (*Access, error) {
	if publicURL == "" {
		return &Access{localOnly: true}, nil
	}

	u, authority, err := parseHTTPSURL(publicURL)
	if err != nil || u.Path != "/tools" || u.RawPath != "" {
		return nil, fmt.Errorf("MCP_PUBLIC_URL は HTTPS の /tools を指定する: %w", ErrInvalidConfig)
	}

	return &Access{authority: authority}, nil
}

func parseHTTPSURL(value string) (*url.URL, string, error) {
	u, err := url.Parse(value)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || strings.ContainsAny(value, "#\r\n\t ") {
		return nil, "", ErrInvalidConfig
	}

	authority, err := canonicalAuthority(u.Host)
	if err != nil {
		return nil, "", err
	}

	return u, authority, nil
}

func canonicalAuthority(host string) (string, error) {
	u, err := url.Parse("https://" + host)
	if err != nil || u.Host != host || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || strings.ContainsAny(host, "\r\n\t %") {
		return "", ErrInvalidConfig
	}

	name := strings.ToLower(u.Hostname())
	if ip := net.ParseIP(name); ip != nil {
		name = ip.String()
	} else {
		if len(name) > 253 {
			return "", ErrInvalidConfig
		}

		for _, label := range strings.Split(name, ".") {
			if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
				return "", ErrInvalidConfig
			}

			for _, c := range label {
				if c != '-' && (c < 'a' || c > 'z') && (c < '0' || c > '9') {
					return "", ErrInvalidConfig
				}
			}
		}
	}

	port := u.Port()
	if port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return "", ErrInvalidConfig
		}

		port = strconv.Itoa(n)
		if port == "443" {
			port = ""
		}
	} else if strings.HasSuffix(host, ":") {
		return "", ErrInvalidConfig
	}

	return name + "|" + port, nil
}

// IsInitialized は New で生成した有効な設定かを返す。nil とゼロ値は false とする。
func (a *Access) IsInitialized() bool { return a != nil && (a.localOnly || a.authority != "") }

// IsPublic は公開 URL を指定して生成した設定かを返す。
// ローカル用の設定、nil、ゼロ値は false とする。
func (a *Access) IsPublic() bool { return a != nil && a.authority != "" }

// AllowsHost は生成時の許可条件に Host が一致するかを返す。
// ローカル用は従来のローカル判定、公開用は authority の正規化と照合を使う。
// 未初期化は false とする。DNS 照会と通信は行わない。
func (a *Access) AllowsHost(host string) bool {
	if !a.IsInitialized() {
		return false
	}

	if a.localOnly {
		return localHost(host)
	}

	authority, err := canonicalAuthority(host)
	return err == nil && a.allowedAuthority(authority)
}

func (a *Access) allowedAuthority(authority string) bool {
	if authority == a.authority {
		return true
	}

	name, _, _ := strings.Cut(authority, "|")
	if name == "localhost" {
		return true
	}

	ip := net.ParseIP(name)
	return ip != nil && ip.IsLoopback()
}

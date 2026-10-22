package mcpaccess

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// ErrInvalidConfig は公開 MCP の URL または保護の組み立てが不正であることを表す。
var ErrInvalidConfig = errors.New("invalid MCP access configuration")

// Access は認証なしの公開 MCP で許可する authority を保持する。
// New で構築し、設定変更時は作り直す。利用者の認証は行わない。
type Access struct{ authority string }

// New は HTTPS の /tools の公開 URL を検査する。
// 空ならローカル専用を表す nil を返す。不正設定の診断へ入力値を含めない。
// 外部通信やバックグラウンド処理は開始しない。
func New(publicURL string) (*Access, error) {
	if publicURL == "" {
		return nil, nil
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

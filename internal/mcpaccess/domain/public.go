package domain

import (
	"net"
	"net/url"
	"strconv"
	"strings"
)

// publicAccess は検査済みの公開 authority を保持する。
type publicAccess struct {
	authority string
}

// IsPublic は公開 URL を持つ設定であることを返す。
func (publicAccess) IsPublic() bool { return true }

// AllowsHost は公開 authority またはローカル Host と一致するかを返す。
// authority を正規化して照合し、DNS 照会や通信は行わない。
func (a publicAccess) AllowsHost(host string) bool {
	authority, err := canonicalAuthority(host)
	return err == nil && a.allowedAuthority(authority)
}

func (a publicAccess) allowedAuthority(authority string) bool {
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

package adapter

import (
	"fmt"
	"strconv"
)

// ListenAddress は PORT を検査し、全インターフェースで待ち受けるアドレスを返す。
// 空は8080とし、不正な値はアドレスを返さず拒否する。
func ListenAddress(port string) (string, error) {
	if port == "" {
		return ":8080", nil
	}
	number, err := strconv.ParseUint(port, 10, 16)
	if err != nil || number == 0 {
		return "", fmt.Errorf("PORT は1〜65535の数字で指定する: %q", port)
	}
	return ":" + strconv.FormatUint(number, 10), nil
}

package main

import (
	"fmt"
	"strconv"
)

func listenAddress(port string) (string, error) {
	if port == "" {
		return ":8080", nil
	}
	number, err := strconv.ParseUint(port, 10, 16)
	if err != nil || number == 0 {
		return "", fmt.Errorf("PORT は1〜65535の数字で指定する: %q", port)
	}
	return ":" + strconv.FormatUint(number, 10), nil
}

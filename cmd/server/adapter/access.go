package adapter

import (
	mcpaccess "github.com/shigaresso/player-lens/internal/mcpaccess/domain"
)

// ReadMCPAccess は公開 URL の環境設定を読み、検査済みのアクセス設定を返す。
// 未指定でもローカル用の Access を生成する。通信は行わない。
func ReadMCPAccess(getenv func(string) string) (mcpaccess.Access, error) {
	return mcpaccess.New(getenv("MCP_PUBLIC_URL"))
}

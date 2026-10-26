package adapter

import reviewmcp "github.com/shigaresso/player-lens/internal/reviewmcp/adapter"

// ReadMCPHTTPConfig は公開 URL の環境設定を読み、生成済みの MCP HTTP 設定を返す。
// 未指定はローカル用とし、通信は行わない。
func ReadMCPHTTPConfig(getenv func(string) string) (reviewmcp.HTTPConfig, error) {
	return reviewmcp.NewHTTPConfig(getenv("MCP_PUBLIC_URL"))
}

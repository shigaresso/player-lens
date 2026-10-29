package adapter

import (
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	accessadapter "github.com/shigaresso/player-lens/internal/mcpaccess/adapter"
	mcpaccess "github.com/shigaresso/player-lens/internal/mcpaccess/domain"
)

// HTTPConfig は MCP の HTTP 入口に渡す許可条件・拒否文言・SDK 設定を保持する。
// NewHTTPConfig で生成し、設定変更時は作り直す。未生成の値は利用側へ渡さない。
// 非公開フィールドに保持し、呼び出し元が個別の値を組み替える API は提供しない。
//
//contractcheck:controlled
type HTTPConfig struct {
	access        mcpaccess.Access
	deniedMessage string
	sdkOptions    mcp.StreamableHTTPOptions
}

// NewHTTPConfig は公開 URL に応じた HTTP 設定を整合した組み合わせで生成する。
// 空 URL はローカル用、指定時は公開用とする。不正 URL はゼロ値とエラーを返す。
// 生成時に通信・時刻取得・利用枠の確保は行わない。
func NewHTTPConfig(publicURL string) (HTTPConfig, error) {
	options := mcp.StreamableHTTPOptions{
		Stateless:                    true,
		JSONResponse:                 true,
		PropagateRequestCancellation: true,
		MaxRequestBodyBytes:          maxRequestBodyBytes,
	}

	if publicURL == "" {
		return HTTPConfig{
			access:        mcpaccess.NewLocal(),
			deniedMessage: "ローカルの Host のみ利用できます",
			sdkOptions:    options,
		}, nil
	}

	access, err := mcpaccess.NewPublic(publicURL)
	if err != nil {
		return HTTPConfig{}, err
	}

	options.DisableLocalhostProtection = true
	return HTTPConfig{
		access:        access,
		deniedMessage: "許可されていない Host です",
		sdkOptions:    options,
	}, nil
}

func (config HTTPConfig) newHandler(getServer func(*http.Request) *mcp.Server) http.Handler {
	handler := mcp.NewStreamableHTTPHandler(getServer, &config.sdkOptions)
	return accessadapter.Protect(config.access, config.deniedMessage, handler)
}

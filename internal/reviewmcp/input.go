// Package reviewmcp はメモリ内のレビュー収集を MCP ツールへ接続する。
package reviewmcp

import (
	"fmt"

	"github.com/shigaresso/player-lens/internal/steam"
)

func parseInput(data []byte) (steam.AppID, steam.CollectionOptions, error) {
	invalid := func() (steam.AppID, steam.CollectionOptions, error) {
		return 0, steam.CollectionOptions{}, fmt.Errorf("invalid MCP collection input: %w", steam.ErrInvalidArgument)
	}
	input, err := decodeToolInput(data)
	if err != nil {
		return invalid()
	}
	appID, options, err := normalizeToolInput(input)
	if err != nil {
		return invalid()
	}
	return appID, options, nil
}

// Package adapter はメモリ内のレビュー収集を MCP ツールへ接続する。
package handler

import (
	"encoding/json"
	"fmt"

	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func parseInput(data []byte) (steamdomain.AppID, steamdomain.CollectionOptions, error) {
	invalid := func() (steamdomain.AppID, steamdomain.CollectionOptions, error) {
		return 0, steamdomain.CollectionOptions{}, fmt.Errorf("invalid MCP collection input: %w", steamdomain.ErrInvalidArgument)
	}
	var input toolInput
	if err := json.Unmarshal(data, &input); err != nil {
		return invalid()
	}

	appID, options, err := normalizeToolInput(input)
	if err != nil {
		return invalid()
	}

	return appID, options, nil
}

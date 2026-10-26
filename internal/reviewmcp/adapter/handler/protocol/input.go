// Package protocol は MCP の入力・応答と公開 schema の表現を担当する。
package protocol

import (
	"encoding/json"
	"fmt"

	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

// ParseInput は MCP 入力を検査・正規化して有効な収集条件を返す。
// 不正時はゼロ値と steamdomain.ErrInvalidArgument を保持したエラーを返す。
func ParseInput(data []byte) (steamdomain.AppID, steamdomain.CollectionOptions, error) {
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

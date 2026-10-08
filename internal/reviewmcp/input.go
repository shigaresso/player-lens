// Package reviewmcp はメモリ内のレビュー収集を MCP ツールへ接続する。
package reviewmcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/shigaresso/player-lens/internal/steam"
)

const (
	defaultReviews = 20
	maxReviews     = 100
	defaultPages   = 3
	maxPages       = 5
)

type toolInput struct {
	AppID      uint32 `json:"app_id"`
	Language   string `json:"language"`
	PageSize   int    `json:"page_size"`
	MaxReviews int    `json:"max_reviews"`
	MaxPages   int    `json:"max_pages"`
}

func parseInput(data []byte) (uint32, steam.CollectionOptions, error) {
	invalid := func() (uint32, steam.CollectionOptions, error) {
		return 0, steam.CollectionOptions{}, fmt.Errorf("invalid MCP collection input: %w", steam.ErrInvalidArgument)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil || fields == nil {
		return invalid()
	}
	for _, value := range fields {
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return invalid()
		}
	}
	var input toolInput
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return invalid()
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return invalid()
	}
	if input.AppID == 0 || input.MaxReviews < 0 || input.MaxReviews > maxReviews || input.MaxPages < 0 || input.MaxPages > maxPages {
		return invalid()
	}
	if input.MaxReviews == 0 {
		input.MaxReviews = defaultReviews
	}
	if input.MaxPages == 0 {
		input.MaxPages = defaultPages
	}
	options, err := steam.NormalizeCollectionOptions(steam.CollectionOptions{
		Language: input.Language, PageSize: input.PageSize, MaxReviews: input.MaxReviews, MaxPages: input.MaxPages,
	})
	if err != nil {
		return invalid()
	}
	return input.AppID, options, nil
}

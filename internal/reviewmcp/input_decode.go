package reviewmcp

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/shigaresso/player-lens/internal/steam"
)

type toolInput struct {
	ReviewType string      `json:"review_type"`
	AppID      steam.AppID `json:"app_id"`
	Language   string      `json:"language"`
	PageSize   int         `json:"page_size"`
	MaxReviews int         `json:"max_reviews"`
	MaxPages   int         `json:"max_pages"`
}

// decodeToolInput は JSON の形式だけを検査し、既定値や取得条件の上限は適用しない。
// 失敗時は部分的に読んだ値を返さず、ゼロ値と ErrInvalidArgument を返す。
func decodeToolInput(data []byte) (toolInput, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil || fields == nil {
		return toolInput{}, steam.ErrInvalidArgument
	}
	for name, value := range fields {
		switch name {
		case "app_id", "review_type", "language", "page_size", "max_reviews", "max_pages":
		default:
			return toolInput{}, steam.ErrInvalidArgument
		}
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return toolInput{}, steam.ErrInvalidArgument
		}
	}
	var input toolInput
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return toolInput{}, steam.ErrInvalidArgument
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return toolInput{}, steam.ErrInvalidArgument
	}
	return input, nil
}

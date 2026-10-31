package protocol

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

type toolInput struct {
	ReviewType string            `json:"review_type"`
	AppID      steamdomain.AppID `json:"app_id"`
	Language   string            `json:"language"`
	PageSize   int               `json:"page_size"`
	MaxReviews int               `json:"max_reviews"`
	MaxPages   int               `json:"max_pages"`
}

// toolInputJSONFields は明示した JSON タグから一度だけ構築し、以後は変更しない。
var toolInputJSONFields = func() map[string]struct{} {
	inputType := reflect.TypeFor[toolInput]()
	fields := make(map[string]struct{}, inputType.NumField())
	for i := 0; i < inputType.NumField(); i++ {
		name, _, _ := strings.Cut(inputType.Field(i).Tag.Get("json"), ",")
		if name != "" && name != "-" {
			fields[name] = struct{}{}
		}
	}

	return fields
}()

// UnmarshalJSON は JSON の形式・正確な項目名・null・値の型を検査する。
// 既定値や取得条件の上限は適用しない。失敗時は ErrInvalidArgument と拒否理由・解析原因を保持し、受信先を変更しない。
// 未知の項目名や入力値を独自の診断文へ転載せず、エラーはクライアントへ直接公開しない。
// 成功時は省略項目を含めて今回の入力だけで受信先を置き換える。
func (input *toolInput) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return fmt.Errorf("decode MCP input fields: %w", errors.Join(steamdomain.ErrInvalidArgument, err))
	}

	if fields == nil {
		return fmt.Errorf("input must be an object: %w", steamdomain.ErrInvalidArgument)
	}

	for name, value := range fields {
		if _, allowed := toolInputJSONFields[name]; !allowed {
			return fmt.Errorf("unknown input field: %w", steamdomain.ErrInvalidArgument)
		}

		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("%s must not be null: %w", name, steamdomain.ErrInvalidArgument)
		}
	}

	// メソッドを持たない別の型へ読み、UnmarshalJSON の再帰呼び出しを避ける。
	type rawToolInput toolInput
	var decoded rawToolInput
	if err := json.Unmarshal(data, &decoded); err != nil {
		return fmt.Errorf("decode MCP input values: %w", errors.Join(steamdomain.ErrInvalidArgument, err))
	}

	*input = toolInput(decoded)
	return nil
}

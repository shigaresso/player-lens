package handler

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestCollectReviewsDefinition(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*mcp.Tool)
	}{
		{"名前と説明の変更をほかの定義へ共有しない", func(tool *mcp.Tool) {
			tool.Name = "changed"
			tool.Description = "変更した説明"
		}},
		{"annotations の変更をほかの定義へ共有しない", func(tool *mcp.Tool) {
			tool.Annotations.ReadOnlyHint = false
			*tool.Annotations.OpenWorldHint = false
		}},
		{"入力 schema の入れ子と必須項目の変更を共有しない", func(tool *mcp.Tool) {
			schema := tool.InputSchema.(map[string]any)
			schema["required"].([]string)[0] = "changed"
			schema["properties"].(map[string]any)["app_id"].(map[string]any)["minimum"] = 2
		}},
		{"出力 schema の入れ子と必須項目の変更を共有しない", func(tool *mcp.Tool) {
			schema := tool.OutputSchema.(map[string]any)
			schema["required"].([]string)[0] = "changed"
			guide := schema["properties"].(map[string]any)["analysis_guide"].(map[string]any)
			guide["properties"].(map[string]any)["version"].(map[string]any)["const"] = "changed"
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			first, second := collectReviewsDefinition(), collectReviewsDefinition()
			before, err := json.Marshal(second)
			if err != nil {
				t.Fatal(err)
			}

			tc.mutate(first)
			after, err := json.Marshal(second)
			if err != nil {
				t.Fatal(err)
			}

			if !bytes.Equal(before, after) {
				t.Fatal("別に生成したツール定義へ変更が伝播した")
			}
		})
	}
}

package protocol

import steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"

// InputSchema は MCP の収集入力の公開 schema を独立した値として生成する。
func InputSchema() map[string]any {
	integer := func(maximum int) map[string]any {
		return map[string]any{"type": "integer", "minimum": 0, "maximum": maximum}
	}
	return map[string]any{
		"type": "object", "additionalProperties": false, "required": []string{"app_id"},
		"properties": map[string]any{
			"app_id":      map[string]any{"type": "integer", "minimum": 1, "maximum": uint64(4294967295)},
			"review_type": map[string]any{"type": "string", "enum": []string{"", "all", "positive", "negative"}, "default": "all", "description": "all は全評価、positive はおすすめ、negative は非おすすめ。省略・空なら all"},
			"language":    map[string]any{"type": "string", "description": "Steam の言語コード。省略・空なら english"},
			"page_size":   integer(steamdomain.MaxPageSize),
			"max_reviews": integer(maxReviews),
			"max_pages":   integer(maxPages),
		},
	}
}

// OutputSchema は MCP の収集結果の公開 schema を独立した値として生成する。
func OutputSchema() map[string]any {
	return map[string]any{
		"type": "object", "additionalProperties": false,
		"required": []string{"collection", "returned_count", "omitted_reviews", "error_code", "message", "analysis_guide"},
		"properties": map[string]any{
			"collection":      map[string]any{"type": "object"},
			"returned_count":  map[string]any{"type": "integer", "minimum": 0},
			"omitted_reviews": map[string]any{"type": "integer", "minimum": 0},
			"error_code":      map[string]any{"type": "string"},
			"message":         map[string]any{"type": "string"},
			"analysis_guide": map[string]any{
				"type": "object", "additionalProperties": false,
				"required": []string{"version", "instructions"},
				"properties": map[string]any{
					"version":      map[string]any{"type": "string", "const": analysisGuideVersion},
					"instructions": map[string]any{"type": "string", "minLength": 1},
				},
			},
		},
	}
}

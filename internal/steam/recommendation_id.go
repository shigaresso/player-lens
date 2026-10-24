package steam

import (
	"fmt"
	"strconv"
)

// RecommendationID は Steam の recommendationid を uint64 の全範囲で表す。
// 0も値として保持し、ID の実在や元の文字列表記は保証しない。
type RecommendationID uint64

func parseRecommendationID(value string) (RecommendationID, error) {
	id, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse Steam recommendationid: %w", err)
	}
	return RecommendationID(id), nil
}

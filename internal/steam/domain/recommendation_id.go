package domain

// RecommendationID は Steam の recommendationid を uint64 の全範囲で表す。
// 0も値として保持し、ID の実在や元の文字列表記は保証しない。
//
//contractcheck:data ゼロを含む uint64 全範囲の値を受け渡す API データ
type RecommendationID uint64

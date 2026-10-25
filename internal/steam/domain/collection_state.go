package domain

import "fmt"

// CollectionState は1回の収集結果と次の取得条件を所有する。
// ゼロ値は使わず NewCollectionState で生成し、1つの収集操作から順番に更新する。
// 通信・context・時刻の取得は行わない。
type CollectionState struct {
	result      Collection
	cursor      string
	seenIDs     map[RecommendationID]bool
	seenCursors map[string]bool
}

// NewCollectionState は正の App ID を検査し、有効な条件で先頭からの収集状態を返す。
// 条件は再検査せずコピーを保持する。App ID が不正なら nil と ErrInvalidArgument の原因を返す。
func NewCollectionState(appID AppID, options CollectionOptions) (*CollectionState, error) {
	if appID == 0 {
		return nil, fmt.Errorf("positive AppID is required: %w", ErrInvalidArgument)
	}

	config := options.Config()
	return &CollectionState{
		result:      Collection{AppID: appID, Options: config, Reviews: []Review{}},
		cursor:      FirstPageCursor,
		seenIDs:     make(map[RecommendationID]bool),
		seenCursors: map[string]bool{FirstPageCursor: true},
	}, nil
}

// NextPage は次の取得条件と続行可否を返す。件数は収集の残件数までに制限する。
// 終了済みならゼロ値と false を返す。
func (s *CollectionState) NextPage() (PageOptions, bool) {
	if s.result.StopReason != "" {
		return PageOptions{}, false
	}

	return PageOptions{
		ReviewType: s.result.Options.ReviewType,
		Language:   s.result.Options.Language,
		Cursor:     s.cursor,
		PageSize:   min(s.result.Options.PageSize, s.result.Options.MaxReviews-len(s.result.Reviews)),
	}, true
}

// Result はレビュー配列も複製した現在の収集結果を返す。
// 戻り値の変更は収集状態へ影響せず、空のレビューも nil にしない。
func (s *CollectionState) Result() Collection {
	result := s.result
	result.Reviews = make([]Review, len(s.result.Reviews))
	copy(result.Reviews, s.result.Reviews)
	return result
}

// AddPage は成功したページを反映し、重複を除いて最初の値と取得順を保持する。
// 空ページ・件数上限・循環・ページ上限の順に判定し、循環は ErrCursorCycle を返す。
// 循環したページも結果へ含める。ページの AppID と Options は採用しない。
// ページは取得 adapter の契約に従う有効な応答を、取得順に渡す。
// 終了済みなら変更せず nil を返す。
func (s *CollectionState) AddPage(page ReviewPage) error {
	if s.result.StopReason != "" {
		return nil
	}

	s.result.PagesFetched++
	if len(page.Reviews) == 0 {
		s.result.StopReason = StopExhausted
		return nil
	}

	for _, review := range page.Reviews {
		if !s.seenIDs[review.RecommendationID] {
			s.seenIDs[review.RecommendationID] = true
			s.result.Reviews = append(s.result.Reviews, review)
		}
	}

	if len(s.result.Reviews) >= s.result.Options.MaxReviews {
		s.result.StopReason = StopReviewLimit
		return nil
	}

	if s.seenCursors[page.NextCursor] {
		s.result.StopReason = StopFailed
		return ErrCursorCycle
	}

	s.seenCursors[page.NextCursor] = true
	s.cursor = page.NextCursor
	if s.result.PagesFetched >= s.result.Options.MaxPages {
		s.result.StopReason = StopPageLimit
	}

	return nil
}

// Fail は進行中の収集を失敗で終了し、成功ページと条件・部分結果を保持する。
// 通信や context の原因は呼び出し側が保持する。終了済みなら変更しない。
func (s *CollectionState) Fail() {
	if s.result.StopReason == "" {
		s.result.StopReason = StopFailed
	}
}

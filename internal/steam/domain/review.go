package domain

// Review は Steam の数値 ID と本文、推薦、作成・更新の UNIX 秒を保持する。
type Review struct {
	RecommendationID RecommendationID
	Text             string           `json:"review"`
	Language         string           `json:"language"`
	Recommended      bool             `json:"voted_up"`
	TimestampCreated TimestampCreated `json:"timestamp_created"`
	TimestampUpdated TimestampUpdated `json:"timestamp_updated"`
}

// ReviewPage は取得条件とレビュー、次のカーソルを保持する。空ページも成功となる。
type ReviewPage struct {
	AppID      AppID
	Options    PageOptions
	Reviews    []Review
	NextCursor string
}

// StopReason は収集の終了理由を表す。上限到達は全件の取得完了を意味しない。
type StopReason string

const (
	// StopExhausted は空ページを取得し、対象条件の走査を終えたことを表す。
	StopExhausted StopReason = "exhausted"
	// StopReviewLimit はレビュー件数の上限で終了したことを表す。
	StopReviewLimit StopReason = "review_limit"
	// StopPageLimit はページ数の上限で終了したことを表す。
	StopPageLimit StopReason = "page_limit"
	// StopFailed は取得や収集の失敗を表す。詳細は返されたエラーで確認する。
	StopFailed StopReason = "failed"
)

// Collection は収集結果を表す。Reviews は同じ ID の最初の本文を取得順に保持する。
// PagesFetched は空ページを含む取得成功ページ数で、失敗したページは含まない。
type Collection struct {
	AppID        AppID
	Options      CollectionOptionsConfig
	Reviews      []Review
	PagesFetched int
	StopReason   StopReason
}

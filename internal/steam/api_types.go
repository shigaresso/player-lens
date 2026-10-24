package steam

// AppID は Steam の appid を uint32 で表す。取得操作は0を拒否する。
type AppID uint32

// TimestampCreated は Steam の timestamp_created を UNIX 秒・uint32 で表す。
type TimestampCreated uint32

// TimestampUpdated は Steam の timestamp_updated を UNIX 秒・uint32 で表す。
type TimestampUpdated uint32

// EUserReviewsReviewType は Steam の全評価・おすすめ・非おすすめを表す int32 の列挙型。
// 内部の取得条件は ReviewType で扱い、HTTP 境界でこの型へ変換する。
type EUserReviewsReviewType int32

const (
	k_EUserReviewsReviewType_All      EUserReviewsReviewType = 0
	k_EUserReviewsReviewType_Positive EUserReviewsReviewType = 1
	k_EUserReviewsReviewType_Negative EUserReviewsReviewType = 2
)

// EUserReviewsAppReviewsFilter は Steam の同名の並び順の列挙値を int32 で表す。
type EUserReviewsAppReviewsFilter int32

// EUserReviewsPurchaseType は Steam の同名の購入種別の列挙値を int32 で表す。
type EUserReviewsPurchaseType int32

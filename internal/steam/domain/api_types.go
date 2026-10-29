package domain

// AppID は Steam の appid を uint32 で表す。取得操作は0を拒否する。
//
//contractcheck:data Steam の uint32 表現であり正数の条件は入力または状態の生成境界で検査する
type AppID uint32

// TimestampCreated は Steam の timestamp_created を UNIX 秒・uint32 で表す。
//
//contractcheck:data Steam の UNIX 秒の uint32 表現を受け渡すデータ
type TimestampCreated uint32

// TimestampUpdated は Steam の timestamp_updated を UNIX 秒・uint32 で表す。
//
//contractcheck:data Steam の UNIX 秒の uint32 表現を受け渡すデータ
type TimestampUpdated uint32

// EUserReviewsReviewType は Steam の全評価・おすすめ・非おすすめを表す int32 の列挙型。
// 内部の取得条件は ReviewType で扱い、HTTP 境界でこの型へ変換する。
//
//contractcheck:data 外部 API の数値表現であり有効性は入力の正規化境界で検査する
type EUserReviewsReviewType int32

const (
	k_EUserReviewsReviewType_All      EUserReviewsReviewType = 0
	k_EUserReviewsReviewType_Positive EUserReviewsReviewType = 1
	k_EUserReviewsReviewType_Negative EUserReviewsReviewType = 2
)

// EUserReviewsAppReviewsFilter は Steam の同名の並び順の列挙値を int32 で表す。
//
//contractcheck:data 外部 API へ送る数値表現を受け渡すデータ
type EUserReviewsAppReviewsFilter int32

// EUserReviewsPurchaseType は Steam の同名の購入種別の列挙値を int32 で表す。
//
//contractcheck:data 外部 API へ送る数値表現を受け渡すデータ
type EUserReviewsPurchaseType int32

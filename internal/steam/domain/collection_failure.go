package domain

import "slices"

// NewFailedCollection は入力条件と取得済み結果を保持した失敗結果のコピーを生成する。
// 入力を変更せず、レビュー配列も複製し、nil と空配列の区別を維持する。
// 終了理由を StopFailed とし、値の検査・既定値の適用・原因の分類は行わない。
func NewFailedCollection(collection Collection) Collection {
	collection.Reviews = slices.Clone(collection.Reviews)
	collection.StopReason = StopFailed
	return collection
}

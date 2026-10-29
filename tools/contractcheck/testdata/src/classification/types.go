package classification

// WithoutPolicy は分類の指定がない。
type WithoutPolicy struct{} // want "生成・更新の分類を指定する"

// WithoutReason は対象外の理由がない。
//
//contractcheck:data
type WithoutReason struct{} // want "対象外の理由を記載する"

// Input は外部入力を保持する。
//
//contractcheck:data 入力として検査する未検証データ
type Input struct{ Number int }

// Reader は生成済みの必須依存を表す。
//
//contractcheck:dependency
type Reader interface{ Read() int }

// Value は生成側で保証する値を表す。
//
//contractcheck:controlled
type Value struct{ number int } // want Value:"controlled in types.go"

// Number は範囲内の数値表現のみを保証する。
//
//contractcheck:data 公開 API の数値表現
type Number uint32

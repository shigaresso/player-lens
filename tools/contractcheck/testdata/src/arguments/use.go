package arguments

import "io"

func Use(value *Value, reader io.Reader) bool {
	if value == nil { // want "必須引数 value の存在を利用側で再検査しない"
		return false
	}
	alias := reader
	if nil != alias { // want "必須引数 reader の存在を利用側で再検査しない"
		return true
	}
	return value.Ready()
}

func Check(value *Value) error {
	return value.Validate() // want "検証済み引数 value を利用側で再検査しない"
}

//contractcheck:optional reader 入力なしでの表示を許可する
func Optional(reader io.Reader, err error) bool {
	return reader != nil && err == nil
}

func Reassigned(reader io.Reader) bool {
	reader = nil
	return reader == nil
}

func Closure(reader io.Reader) func() bool {
	return func() bool { return reader == nil } // want "必須引数 reader の存在を利用側で再検査しない"
}

//contractcheck:optional reader
func MissingReason(reader io.Reader) { // want "任意引数の理由を記載する"
}

//contractcheck:optional missing その名前の引数がない
func Unknown(reader io.Reader) { // want "任意引数 missing が存在しない"
}

//contractcheck:constructor
func FakeConstructor(reader io.Reader) bool { // want "生成関数は定義ファイルの対象型を返す"
	return reader == nil // want "必須引数 reader の存在を利用側で再検査しない"
}

func Switch(reader io.Reader) bool {
	switch reader {
	case nil: // want "必須引数 reader の存在を利用側で再検査しない"
		return false
	}
	return true
}

func SelfAssignment(reader io.Reader) bool {
	reader = reader
	return reader == nil // want "必須引数 reader の存在を利用側で再検査しない"
}

func MethodValue(value *Value) func() error {
	return value.Validate // want "検証済み引数 value を利用側で再検査しない"
}

var Callback = func(reader io.Reader) bool {
	return reader == nil // want "必須引数 reader の存在を利用側で再検査しない"
}

func TypedNil(reader io.Reader) bool {
	return reader == io.Reader(nil) // want "必須引数 reader の存在を利用側で再検査しない"
}

//contractcheck:optional value 省略可能な依存を受け取る
func OptionalValidation(value *Value) error {
	if value == nil {
		return nil
	}
	return value.Validate() // want "検証済み引数 value を利用側で再検査しない"
}

//contractcheck:optional number 整数は nil にできない
func InvalidOptional(number int) { // want "任意引数 number は nil を許容する型とする"
}

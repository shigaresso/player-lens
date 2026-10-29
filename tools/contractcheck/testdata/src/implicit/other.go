package implicit

type Settings struct {
	Value Value
	Count int
}

func Local() Value {
	var value Value // want "Value の生成は定義ファイル value.go に限定する"
	return value
}

var Global Value // want "Value の生成は定義ファイル value.go に限定する"

func Pointer() *Value { return new(Value) } // want "Value の生成は定義ファイル value.go に限定する"

func Nested() Settings { return Settings{} } // want "Value の生成は定義ファイル value.go に限定する"

func Array() [2]Value { return [2]Value{New()} } // want "Value の生成は定義ファイル value.go に限定する"

func Slice() []Value { return make([]Value, 2) } // want "Value の生成は定義ファイル value.go に限定する"

func PointerSlice(n int) []*Value { return make([]*Value, n) } // want "Value の生成は定義ファイル value.go に限定する"

func Good() (Settings, [1]Value, []Value) {
	return Settings{Value: New()}, [1]Value{New()}, make([]Value, 0)
}

func NamedResult() (value Value) { // want "Value の生成は定義ファイル value.go に限定する"
	value = New()
	return
}

type Shape struct{ number int }

func Convert() Value { return Value(Shape{}) } // want "Value の生成は定義ファイル value.go に限定する"

func PointerDeclaration() *Value {
	var value *Value // want "Value の生成は定義ファイル value.go に限定する"
	return value
}

var ExplicitNil *Value = nil // want "Value の生成は定義ファイル value.go に限定する"

func ReturnNil() *Value { return nil } // want "Value の生成は定義ファイル value.go に限定する"

func ConvertNil() *Value { return (*Value)(nil) } // want "Value の生成は定義ファイル value.go に限定する"

func NilElements() []*Value { return []*Value{nil} } // want "Value の生成は定義ファイル value.go に限定する"

func NilField() struct{ Value *Value } {
	return struct{ Value *Value }{Value: nil} // want "Value の生成は定義ファイル value.go に限定する"
}

func Map(values map[string]Value) Value {
	return values["unknown"] // want "Value の取得は存在確認用の bool と一緒に受け取る"
}

func MapChecked(values map[string]Value) (Value, bool) {
	value, ok := values["key"]
	return value, ok
}

func Channel(values <-chan Value) Value {
	return <-values // want "Value の取得は存在確認用の bool と一緒に受け取る"
}

package direct

// Value は生成関数を通して利用する。
//
//contractcheck:controlled
type Value struct{ number int } // want Value:"controlled in value.go"

func New() Value { return Value{number: 1} }

func Copy(value Value) Value { return value }

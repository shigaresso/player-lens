package implicit

//contractcheck:controlled
type Value struct{ number int } // want Value:"controlled in value.go"

func New() Value { return Value{number: 1} }

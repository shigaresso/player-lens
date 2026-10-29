package arguments

//contractcheck:controlled
type Value struct{ number int } // want Value:"controlled in value.go"

//contractcheck:constructor
func New(input *int) *Value {
	if input == nil {
		return &Value{}
	}
	return &Value{number: *input}
}

func (v *Value) Validate() error { return nil }
func (v *Value) Ready() bool     { return v.number > 0 }

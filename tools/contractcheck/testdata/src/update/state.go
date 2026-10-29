package update

//contractcheck:controlled
type State struct { // want State:"controlled in state.go"
	number int
	values []int
}

func New() *State { return &State{values: []int{1}} }

func (s *State) Add() { s.number++ }

func (s *State) Snapshot() []int { return append([]int(nil), s.values...) }

func (s *State) Leak() []int { return s.values } // want "State の内部参照をそのまま返さない"

func (s *State) AliasLeak() []int {
	values := s.values
	return values // want "State の内部参照をそのまま返さない"
}

func (s *State) AppendLeak() []int { return append(s.values, 2) } // want "State の内部参照をそのまま返さない"

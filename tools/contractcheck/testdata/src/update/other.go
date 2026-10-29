package update

func Bad(s *State) {
	s.number++      // want "State の更新は定義ファイル state.go に限定する"
	s.number = 3    // want "State の更新は定義ファイル state.go に限定する"
	s.values[0] = 4 // want "State の更新は定義ファイル state.go に限定する"
	_ = &s.number   // want "State の内部への参照取得は定義ファイル state.go に限定する"
	*s = *New()     // want "State の更新は定義ファイル state.go に限定する"
	s = nil         // want "State の生成は定義ファイル state.go に限定する"
}

func Aliased(s *State) {
	values := s.values // want "State の内部への参照取得は定義ファイル state.go に限定する"
	values[0] = 0
}

func Good(s *State) int { s.Add(); return s.number }

type representation State

func ConvertPointer(s *State) {
	(*representation)(s).number = 0 // want "State の内部への参照取得は定義ファイル state.go に限定する"
}

func consume([]int) {}

func Borrow(s *State) {
	consume(s.values)        // want "State の内部への参照取得は定義ファイル state.go に限定する"
	copy(s.values, []int{2}) // want "State の内部への参照取得は定義ファイル state.go に限定する"
	_ = len(s.values)
}

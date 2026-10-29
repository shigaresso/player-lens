package consumer

import "direct"

func Bad() direct.Value { return direct.Value{} } // want "Value の生成は定義ファイル value.go に限定する"

type Value struct{ Number int }

func OtherType() Value { return Value{} }

func Good(value direct.Value) direct.Value { return value }

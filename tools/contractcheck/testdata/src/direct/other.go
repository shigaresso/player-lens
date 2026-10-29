package direct

type Alias = Value

func Bad() Value { return Value{} } // want "Value の生成は定義ファイル value.go に限定する"

func AliasBad() Alias { return Alias{} } // want "Value の生成は定義ファイル value.go に限定する"

func Good() Value { return Copy(New()) }

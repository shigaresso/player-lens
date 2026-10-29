package declaration

//contractcheck:controlled
type Value struct { // want "対象型のフィールドは非公開にする"
	Number int
}

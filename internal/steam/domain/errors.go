package domain

import "errors"

// ErrInvalidArgument は通信前に入力または接続設定が拒否されたことを表す。
var ErrInvalidArgument = errors.New("invalid Steam review argument")

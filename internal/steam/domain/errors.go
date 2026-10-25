package domain

import "errors"

// ErrInvalidArgument は通信前に入力または接続設定が拒否されたことを表す。
var ErrInvalidArgument = errors.New("invalid Steam review argument")

// ErrCursorCycle は収集中に取得済みカーソルへの循環を検出したことを表す。
var ErrCursorCycle = errors.New("steam review cursor cycle")

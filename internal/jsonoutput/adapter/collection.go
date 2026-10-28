// Package adapter は収集結果の JSON 変換と writer への出力を担当する。
package adapter

import (
	"encoding/json"
	"fmt"

	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

// MarshalCollection は通信・保存を行わず schema_version=1 の JSON を返す。
// 入力を変更せず、空のレビューは [] とする。時刻は元データの UNIX 秒である。
// 取得時刻・永続的な本文版・分析対象集合はこの出力形式では管理しない。
func MarshalCollection(result steamdomain.Collection) ([]byte, error) {
	document, err := newCollectionDocument(result)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("encode review collection JSON: %w", err)
	}

	return data, nil
}

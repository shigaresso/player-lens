// Package jsonoutput は収集結果の JSON 変換と writer への出力を担当する。
package jsonoutput

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/shigaresso/player-lens/internal/steam"
)

type collectionDocument struct {
	SchemaVersion int               `json:"schema_version"`
	AppID         steam.AppID       `json:"app_id"`
	Options       collectionOptions `json:"options"`
	Reviews       []reviewDocument  `json:"reviews"`
	PagesFetched  int               `json:"pages_fetched"`
	StopReason    steam.StopReason  `json:"stop_reason"`
}

type collectionOptions struct {
	ReviewType steam.ReviewType `json:"review_type"`
	Language   string           `json:"language"`
	PageSize   int              `json:"page_size"`
	MaxReviews int              `json:"max_reviews"`
	MaxPages   int              `json:"max_pages"`
}

type reviewDocument struct {
	RecommendationID string                 `json:"id"`
	Text             string                 `json:"text"`
	Language         string                 `json:"language"`
	Recommended      bool                   `json:"recommended"`
	TimestampCreated steam.TimestampCreated `json:"created_at"`
	TimestampUpdated steam.TimestampUpdated `json:"updated_at"`
}

// MarshalCollection は通信・保存を行わず schema_version=1 の JSON を返す。
// 入力を変更せず、空のレビューは [] とする。時刻は元データの UNIX 秒である。
// 取得時刻・永続的な本文版・分析対象集合はこの出力形式では管理しない。
func MarshalCollection(result steam.Collection) ([]byte, error) {
	reviews := make([]reviewDocument, len(result.Reviews))
	for i, review := range result.Reviews {
		reviews[i] = reviewDocument{strconv.FormatUint(uint64(review.RecommendationID), 10), review.Text, review.Language, review.Recommended, review.TimestampCreated, review.TimestampUpdated}
	}
	document := collectionDocument{
		SchemaVersion: 1,
		AppID:         result.AppID,
		Options:       collectionOptions{result.Options.ReviewType, result.Options.Language, result.Options.PageSize, result.Options.MaxReviews, result.Options.MaxPages},
		Reviews:       reviews,
		PagesFetched:  result.PagesFetched,
		StopReason:    result.StopReason,
	}
	data, err := json.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("encode review collection JSON: %w", err)
	}
	return data, nil
}

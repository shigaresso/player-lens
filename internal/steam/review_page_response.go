package steam

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
)

var reviewIDPattern = regexp.MustCompile(`^[0-9]+$`)

type reviewPageResponse struct {
	Response *struct {
		Reviews []Review `json:"reviews"`
		Cursor  string   `json:"cursor"`
	} `json:"response"`
}

// validReviewID は符号なしの数字列で表した正の uint64 の ID だけを受け付ける。
func validReviewID(value string) bool {
	id, err := strconv.ParseUint(value, 10, 64)
	return err == nil && id > 0 && reviewIDPattern.MatchString(value)
}

// decodeReviewPage は正規化済みの条件と本文からページを作り、不正な応答ではゼロ値を返す。
// Body の読み取り・サイズ制限・解放は呼び出し側が担当する。
func decodeReviewPage(appID uint32, options PageOptions, body []byte) (ReviewPage, error) {
	var result reviewPageResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return ReviewPage{}, fmt.Errorf("decode Steam review response: %w", errors.Join(ErrInvalidResponse, err))
	}
	if result.Response == nil || len(result.Response.Reviews) > options.PageSize {
		return ReviewPage{}, fmt.Errorf("missing response or too many reviews: %w", ErrInvalidResponse)
	}
	if len(result.Response.Reviews) > 0 && result.Response.Cursor == "" {
		return ReviewPage{}, fmt.Errorf("missing next cursor: %w", ErrInvalidResponse)
	}
	for _, review := range result.Response.Reviews {
		if !validReviewID(review.ID) {
			return ReviewPage{}, fmt.Errorf("invalid review ID: %w", ErrInvalidResponse)
		}
	}
	if result.Response.Reviews == nil {
		result.Response.Reviews = []Review{}
	}
	return ReviewPage{AppID: appID, Options: options, Reviews: result.Response.Reviews, NextCursor: result.Response.Cursor}, nil
}

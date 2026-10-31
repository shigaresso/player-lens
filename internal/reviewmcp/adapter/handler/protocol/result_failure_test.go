package protocol

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/shigaresso/player-lens/internal/reviewmcp/adapter/internal/testutil"
	mcpapplication "github.com/shigaresso/player-lens/internal/reviewmcp/application"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestEncodeCollectionResultEncodingFailureCause(t *testing.T) {
	cause := errors.New("secret: upstream details")
	for _, tc := range []struct {
		name       string
		collection steamdomain.Collection
		wantErr    error
	}{
		{"省略後も大きい場合に変換と収集の原因を保持する", steamdomain.Collection{Options: steamdomain.CollectionOptionsConfig{Language: strings.Repeat("x", maxResultBytes+1)}}, ErrResponseTooLarge},
		{"JSON変換が失敗した場合も変換と収集の原因を保持する", steamdomain.Collection{Options: steamdomain.CollectionOptionsConfig{ReviewType: steamdomain.ReviewType(3)}}, steamdomain.ErrInvalidArgument},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := EncodeResult(mcpapplication.Result{Collection: tc.collection, Err: cause})
			if result != nil || !errors.Is(err, tc.wantErr) {
				t.Fatalf("result=%+v error=%v", result, err)
			}

			if !errors.Is(err, cause) {
				t.Fatalf("収集の原因を失った: %v", err)
			}

			if strings.Contains(err.Error(), "secret") {
				t.Fatalf("内部の収集原因を表示文言に含めた: %v", err)
			}
		})
	}
}

func TestInvalidInputResultCause(t *testing.T) {
	cause := errors.New("private: input diagnostic")
	result, err := InvalidInputResult(cause)
	if err != nil || result == nil || !result.IsError || !errors.Is(result.GetError(), cause) {
		t.Fatalf("入力原因を内部に保持しなかった: result=%+v err=%v", result, err)
	}

	out := testutil.DecodeOutput(t, result)
	if out.ErrorCode != "invalid_input" || out.Message != "取得条件が不正です。app_id とツールの入力上限を確認してください。" || out.Collection.AppID != 0 || out.ReturnedCount != 0 || out.Collection.Reviews == nil || len(out.Collection.Reviews) != 0 {
		t.Fatalf("入力不正の公開応答を変更した: %+v", out)
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(data), "private") {
		t.Fatal("SDK の JSON 応答に入力原因を含めた")
	}
}

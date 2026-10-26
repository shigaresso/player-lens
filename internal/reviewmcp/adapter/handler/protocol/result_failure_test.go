package protocol

import (
	"errors"
	"strings"
	"testing"

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
			result, err := encodeCollectionResult(tc.collection, cause)
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

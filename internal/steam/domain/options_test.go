package domain_test

import (
	"errors"
	"strings"
	"testing"

	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestNormalizePageOptions(t *testing.T) {
	tests := []struct {
		name    string
		input   steamdomain.PageOptions
		want    steamdomain.PageOptions
		invalid bool
	}{
		{"未指定を英語と先頭と20件へ補完する", steamdomain.PageOptions{}, steamdomain.PageOptions{Language: "english", Cursor: "*", PageSize: 20}, false},
		{"指定した件数を保って他の既定値を補完する", steamdomain.PageOptions{PageSize: 1}, steamdomain.PageOptions{Language: "english", Cursor: "*", PageSize: 1}, false},
		{"指定した言語とカーソルを保って20件へ補完する", steamdomain.PageOptions{Language: "japanese", Cursor: "次+/="}, steamdomain.PageOptions{Language: "japanese", Cursor: "次+/=", PageSize: 20}, false},
		{"最大件数と言語とカーソルの境界を保持する", steamdomain.PageOptions{Language: strings.Repeat("a", 32), Cursor: strings.Repeat("x", 4096), PageSize: 100}, steamdomain.PageOptions{Language: strings.Repeat("a", 32), Cursor: strings.Repeat("x", 4096), PageSize: 100}, false},
		{"全言語と最小件数を保持する", steamdomain.PageOptions{Language: "all", Cursor: steamdomain.FirstPageCursor, PageSize: 1}, steamdomain.PageOptions{Language: "all", Cursor: "*", PageSize: 1}, false},
		{"負の件数を拒否する", steamdomain.PageOptions{PageSize: -1}, steamdomain.PageOptions{}, true},
		{"101件を拒否する", steamdomain.PageOptions{PageSize: 101}, steamdomain.PageOptions{}, true},
		{"大文字の言語を拒否する", steamdomain.PageOptions{Language: "English"}, steamdomain.PageOptions{}, true},
		{"空白を含む言語を拒否する", steamdomain.PageOptions{Language: "en glish"}, steamdomain.PageOptions{}, true},
		{"数字で始まる言語を拒否する", steamdomain.PageOptions{Language: "1english"}, steamdomain.PageOptions{}, true},
		{"33文字の言語を拒否する", steamdomain.PageOptions{Language: strings.Repeat("a", 33)}, steamdomain.PageOptions{}, true},
		{"4097バイトのカーソルを拒否する", steamdomain.PageOptions{Cursor: strings.Repeat("x", 4097)}, steamdomain.PageOptions{}, true},
		{"Unicodeのカーソルもバイト長で拒否する", steamdomain.PageOptions{Cursor: strings.Repeat("次", 1366)}, steamdomain.PageOptions{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.invalid {
				tt.want.ReviewType = steamdomain.ReviewTypeAll
			}

			before := tt.input
			got, err := steamdomain.NormalizePageOptions(tt.input)
			if tt.invalid {
				if !errors.Is(err, steamdomain.ErrInvalidArgument) {
					t.Fatalf("error=%v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}

			if got != tt.want {
				t.Errorf("options=%#v want=%#v", got, tt.want)
			}

			if tt.input != before {
				t.Errorf("入力を変更した: %#v", tt.input)
			}

			if !tt.invalid {
				again, err := steamdomain.NormalizePageOptions(got)
				if err != nil || again != got {
					t.Errorf("再適用で値を変えた: %#v error=%v", again, err)
				}
			}
		})
	}
}

func TestMaxPageSize(t *testing.T) {
	if steamdomain.MaxPageSize != 100 {
		t.Fatalf("Steamの最大件数=%d want=100", steamdomain.MaxPageSize)
	}
}

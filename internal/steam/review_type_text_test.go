package steam_test

import (
	"errors"
	"testing"

	"github.com/shigaresso/player-lens/internal/steam"
)

func TestParseReviewType(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		want        steam.ReviewType
		invalid     bool
	}{
		{"省略は全評価の0になる", "", 0, false},
		{"全評価は0になる", "all", 0, false},
		{"おすすめは1になる", "positive", 1, false},
		{"非おすすめは2になる", "negative", 2, false},
		{"未知の文字列を拒否する", "mixed", 0, true},
		{"大文字を拒否する", "Positive", 0, true},
		{"空白付きの値を拒否する", " negative ", 0, true},
		{"数値文字列を拒否する", "1", 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := steam.ParseReviewType(tc.input)
			if got != tc.want || (err != nil) != tc.invalid || errors.Is(err, steam.ErrInvalidArgument) != tc.invalid {
				t.Fatalf("kind=%d error=%v want=%d invalid=%v", got, err, tc.want, tc.invalid)
			}
		})
	}
}

func TestFormatReviewType(t *testing.T) {
	for _, tc := range []struct {
		name    string
		input   steam.ReviewType
		want    string
		invalid bool
	}{
		{"0は全評価の文字列になる", 0, "all", false},
		{"1はおすすめの文字列になる", 1, "positive", false},
		{"2は非おすすめの文字列になる", 2, "negative", false},
		{"負の数値を拒否する", -1, "", true},
		{"未定義の3を拒否する", 3, "", true},
		{"int32最大値を拒否する", 2147483647, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := steam.FormatReviewType(tc.input)
			if got != tc.want || (err != nil) != tc.invalid || errors.Is(err, steam.ErrInvalidArgument) != tc.invalid {
				t.Fatalf("name=%q error=%v want=%q invalid=%v", got, err, tc.want, tc.invalid)
			}
		})
	}
}

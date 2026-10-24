package reviewmcp

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/shigaresso/player-lens/internal/steam"
)

func TestToolInputJSONDecoding(t *testing.T) {
	for _, tc := range []struct {
		name    string
		input   string
		want    toolInput
		invalid bool
	}{
		{"正確な項目名を読み取り既定値は適用しない", `{"app_id":440,"review_type":"negative"}`, toolInput{AppID: 440, ReviewType: "negative"}, false},
		{"未知の項目を無視せず拒否する", `{"app_id":440,"cursor":"*"}`, toolInput{}, true},
		{"大文字の項目名を別表記として許可しない", `{"APP_ID":440}`, toolInput{}, true},
		{"null オブジェクトを拒否する", `null`, toolInput{}, true},
		{"null の項目をゼロ値として許可しない", `{"app_id":440,"language":null}`, toolInput{}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got toolInput
			err := json.Unmarshal([]byte(tc.input), &got)
			if tc.invalid {
				if !errors.Is(err, steam.ErrInvalidArgument) || got != (toolInput{}) {
					t.Fatalf("input = %+v, err = %v", got, err)
				}
			} else if err != nil || got != tc.want {
				t.Fatalf("input = %+v, err = %v; want %+v", got, err, tc.want)
			}
		})
	}
}

func TestToolInputJSONReceiver(t *testing.T) {
	initial := toolInput{AppID: 440, ReviewType: "positive", Language: "japanese", PageSize: 10, MaxReviews: 20, MaxPages: 2}
	for _, tc := range []struct {
		name    string
		input   string
		want    toolInput
		invalid bool
	}{
		{"成功時は省略項目を以前の値で補わない", `{"app_id":570}`, toolInput{AppID: 570}, false},
		{"空オブジェクトで以前の値をすべて置き換える", `{}`, toolInput{}, false},
		{"未知項目の拒否時は受信先を変更しない", `{"app_id":570,"cursor":"*"}`, initial, true},
		{"大文字の拒否時は受信先を変更しない", `{"APP_ID":570}`, initial, true},
		{"null の拒否時は受信先を変更しない", `null`, initial, true},
		{"途中の型違いで受信先を部分的に書き換えない", `{"app_id":570,"max_pages":"bad"}`, initial, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := initial
			err := json.Unmarshal([]byte(tc.input), &got)
			if tc.invalid {
				if !errors.Is(err, steam.ErrInvalidArgument) {
					t.Fatalf("err = %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("input = %+v; want %+v", got, tc.want)
			}
		})
	}
}

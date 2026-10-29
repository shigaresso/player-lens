package adapter_test

import (
	"reflect"
	"strings"
	"testing"

	serveradapter "github.com/shigaresso/player-lens/cmd/server/adapter"
)

func TestNewValidatedPortNumber(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "未指定なら既定の8080を返す", input: "", want: "8080"},
		{name: "指定したポート番号を返す", input: "10000", want: "10000"},
		{name: "下限の1を受け付ける", input: "1", want: "1"},
		{name: "上限の65535を受け付ける", input: "65535", want: "65535"},
		{name: "先頭のゼロを正規化する", input: "00080", want: "80"},
		{name: "ゼロを拒否する", input: "0", wantErr: true},
		{name: "ゼロだけの文字列を拒否する", input: "000", wantErr: true},
		{name: "上限超過を拒否する", input: "65536", wantErr: true},
		{name: "数値変換の範囲を超える入力を拒否する", input: "18446744073709551616", wantErr: true},
		{name: "負数を拒否する", input: "-1", wantErr: true},
		{name: "正の符号を拒否する", input: "+8080", wantErr: true},
		{name: "前方の空白を拒否する", input: " 8080", wantErr: true},
		{name: "後方の空白を拒否する", input: "8080 ", wantErr: true},
		{name: "改行を拒否する", input: "8080\n", wantErr: true},
		{name: "数字以外を拒否する", input: "http", wantErr: true},
		{name: "全角数字を拒否する", input: "８０８０", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := serveradapter.NewValidatedPortNumber(tt.input)
			if tt.wantErr {
				if err == nil || !strings.Contains(err.Error(), "PORT は1〜65535の数字で指定する:") || !reflect.ValueOf(got).IsZero() {
					t.Fatalf("不正な入力ではゼロ値とPORTの説明を返す: port=%v err=%v", got, err)
				}

				return
			}

			if err != nil {
				t.Fatalf("有効な入力を拒否した: %v", err)
			}

			if got.String() != tt.want {
				t.Fatalf("port=%q; want=%q", got.String(), tt.want)
			}
		})
	}
}

func TestValidatedPortNumberDefaults(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "未指定の入力はゼロ値と同じ既定値となる", input: ""},
		{name: "明示した8080もゼロ値と同じ既定値となる", input: "8080"},
		{name: "先頭にゼロがある8080も同じ既定値となる", input: "08080"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defaultPort, err := serveradapter.NewValidatedPortNumber("")
			if err != nil {
				t.Fatal(err)
			}

			got, err := serveradapter.NewValidatedPortNumber(tt.input)
			if err != nil {
				t.Fatal(err)
			}

			if defaultPort.String() != "8080" || got != defaultPort || got.String() != "8080" {
				t.Fatalf("既定値が一致しない: zero=%v port=%v err=%v", defaultPort, got, err)
			}
		})
	}
}

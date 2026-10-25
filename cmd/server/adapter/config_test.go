package adapter

import "testing"

func TestListenAddress(t *testing.T) {
	tests := []struct {
		name string
		port string
		want string
	}{
		{"未指定なら8080で待ち受ける", "", ":8080"},
		{"指定ポートで全インターフェースを待ち受ける", "10000", ":10000"},
		{"下限を受け付ける", "1", ":1"},
		{"上限を受け付ける", "65535", ":65535"},
		{"先頭のゼロを正規化する", "08080", ":8080"},
		{"ゼロを拒否する", "0", ""},
		{"上限超過を拒否する", "65536", ""},
		{"負数を拒否する", "-1", ""},
		{"符号を拒否する", "+8080", ""},
		{"空白を拒否する", " 8080", ""},
		{"非数字を拒否する", "http", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ListenAddress(tt.port)
			if tt.want == "" {
				if err == nil || got != "" {
					t.Fatalf("不正値を受け付けた: address=%q err=%v", got, err)
				}

				return
			}

			if err != nil || got != tt.want {
				t.Fatalf("address=%q err=%v; want=%q", got, err, tt.want)
			}
		})
	}
}

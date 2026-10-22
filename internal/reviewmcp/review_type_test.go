package reviewmcp

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/shigaresso/player-lens/internal/steam"
)

func TestParseInputReviewType(t *testing.T) {
	for _, tt := range []struct {
		name    string
		input   string
		want    steam.ReviewType
		invalid bool
	}{
		{"省略は全評価", `{"app_id":440}`, steam.ReviewTypeAll, false},
		{"空は全評価", `{"app_id":440,"review_type":""}`, steam.ReviewTypeAll, false},
		{"全評価を明示する", `{"app_id":440,"review_type":"all"}`, steam.ReviewTypeAll, false},
		{"おすすめを指定する", `{"app_id":440,"review_type":"positive"}`, steam.ReviewTypePositive, false},
		{"非おすすめを指定する", `{"app_id":440,"review_type":"negative"}`, steam.ReviewTypeNegative, false},
		{"未知値を拒否する", `{"app_id":440,"review_type":"mixed"}`, "", true},
		{"大文字を拒否する", `{"app_id":440,"review_type":"Positive"}`, "", true},
		{"前後空白を拒否する", `{"app_id":440,"review_type":" negative "}`, "", true},
		{"nullを拒否する", `{"app_id":440,"review_type":null}`, "", true},
		{"数値を拒否する", `{"app_id":440,"review_type":1}`, "", true},
		{"配列を拒否する", `{"app_id":440,"review_type":[]}`, "", true},
		{"真偽値を拒否する", `{"app_id":440,"review_type":true}`, "", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			id, options, err := parseInput([]byte(tt.input))
			if tt.invalid {
				if err == nil || id != 0 || options != (steam.CollectionOptions{}) {
					t.Fatalf("id=%d options=%+v err=%v", id, options, err)
				}
			} else if err != nil || id != 440 || options.ReviewType != tt.want {
				t.Fatalf("id=%d options=%+v err=%v", id, options, err)
			}
		})
	}
}

func TestInputSchemaReviewType(t *testing.T) {
	data, err := json.Marshal(inputSchema())
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Required   []string `json:"required"`
		Properties map[string]struct {
			Type        string   `json:"type"`
			Enum        []string `json:"enum"`
			Default     string   `json:"default"`
			Description string   `json:"description"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	field := schema.Properties["review_type"]
	if field.Type != "string" || field.Default != "all" || field.Description == "" || !reflect.DeepEqual(field.Enum, []string{"", "all", "positive", "negative"}) || !reflect.DeepEqual(schema.Required, []string{"app_id"}) {
		t.Fatalf("schema=%s", data)
	}
}

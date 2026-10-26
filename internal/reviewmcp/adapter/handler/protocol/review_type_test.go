package protocol

import (
	"encoding/json"
	"reflect"
	"testing"

	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestParseInputReviewType(t *testing.T) {
	for _, tt := range []struct {
		name    string
		input   string
		want    steamdomain.ReviewType
		invalid bool
	}{
		{"省略は全評価", `{"app_id":440}`, steamdomain.ReviewTypeAll, false},
		{"空は全評価", `{"app_id":440,"review_type":""}`, steamdomain.ReviewTypeAll, false},
		{"全評価を明示する", `{"app_id":440,"review_type":"all"}`, steamdomain.ReviewTypeAll, false},
		{"おすすめを指定する", `{"app_id":440,"review_type":"positive"}`, steamdomain.ReviewTypePositive, false},
		{"非おすすめを指定する", `{"app_id":440,"review_type":"negative"}`, steamdomain.ReviewTypeNegative, false},
		{"未知値を拒否する", `{"app_id":440,"review_type":"mixed"}`, 0, true},
		{"大文字を拒否する", `{"app_id":440,"review_type":"Positive"}`, 0, true},
		{"前後空白を拒否する", `{"app_id":440,"review_type":" negative "}`, 0, true},
		{"nullを拒否する", `{"app_id":440,"review_type":null}`, 0, true},
		{"数値を拒否する", `{"app_id":440,"review_type":1}`, 0, true},
		{"配列を拒否する", `{"app_id":440,"review_type":[]}`, 0, true},
		{"真偽値を拒否する", `{"app_id":440,"review_type":true}`, 0, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			id, options, err := ParseInput([]byte(tt.input))
			if tt.invalid {
				if err == nil || id != 0 || options != (steamdomain.CollectionOptions{}) {
					t.Fatalf("id=%d options=%+v err=%v", id, options, err)
				}
			} else if err != nil || id != 440 || options.Config().ReviewType != tt.want {
				t.Fatalf("id=%d options=%+v err=%v", id, options, err)
			}
		})
	}
}

func TestInputSchemaReviewType(t *testing.T) {
	data, err := json.Marshal(InputSchema())
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

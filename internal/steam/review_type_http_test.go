package steam_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/shigaresso/player-lens/internal/steam"
)

func TestFetchReviewsPageReviewType(t *testing.T) {
	for _, tt := range []struct {
		name        string
		input       steam.EUserReviewsReviewType
		wantType    int
		recommended bool
	}{
		{"省略は全評価を要求する", 0, 0, true},
		{"全評価を明示できる", steam.ReviewTypeAll, 0, false},
		{"おすすめだけを要求する", steam.ReviewTypePositive, 1, true},
		{"非おすすめだけを要求する", steam.ReviewTypeNegative, 2, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			client, err := steam.NewClient(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				var input struct {
					ReviewType *int `json:"review_type"`
				}
				if err := json.Unmarshal([]byte(r.URL.Query().Get("input_json")), &input); err != nil {
					t.Fatal(err)
				}
				if input.ReviewType == nil || *input.ReviewType != tt.wantType {
					t.Errorf("review_type=%v want=%d", input.ReviewType, tt.wantType)
				}
				body := `{"response":{"reviews":[{"recommendationid":"123","voted_up":false}],"cursor":"next"}}`
				if tt.recommended {
					body = strings.Replace(body, "false", "true", 1)
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
			})}, "")
			if err != nil {
				t.Fatal(err)
			}
			got, err := client.FetchReviewsPage(t.Context(), 440, steam.PageOptions{ReviewType: tt.input, PageSize: 1})
			if err != nil || calls != 1 || len(got.Reviews) != 1 || got.Reviews[0].Recommended != tt.recommended {
				t.Fatalf("page=%+v calls=%d err=%v", got, calls, err)
			}
			want := tt.input
			if got.Options.ReviewType != want {
				t.Fatalf("条件を失った: %+v", got.Options)
			}
		})
	}
}

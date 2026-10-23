package reviewcli_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/shigaresso/player-lens/internal/jsonoutput"
	"github.com/shigaresso/player-lens/internal/reviewcli"
	"github.com/shigaresso/player-lens/internal/steam"
)

func TestRunNumericRecommendationIDs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := fmt.Fprint(w, `{"response":{"reviews":[{"recommendationid":"0000"},{"recommendationid":"18446744073709551615"}],"cursor":"next"}}`); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	client, err := steam.NewClient(server.Client(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	saver, err := jsonoutput.NewSaver(&output)
	if err != nil {
		t.Fatal(err)
	}
	if err := reviewcli.Run(t.Context(), []string{"--app-id=4294967295", "--max-reviews=2"}, client, saver); err != nil {
		t.Fatal(err)
	}
	var got struct {
		AppID   uint32 `json:"app_id"`
		Reviews []struct {
			ID string `json:"id"`
		} `json:"reviews"`
		StopReason string `json:"stop_reason"`
	}
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.AppID != 4294967295 || got.StopReason != "review_limit" || len(got.Reviews) != 2 {
		t.Fatalf("JSON=%s", output.Bytes())
	}
	if ids := []string{got.Reviews[0].ID, got.Reviews[1].ID}; !reflect.DeepEqual(ids, []string{"0", "18446744073709551615"}) {
		t.Fatalf("ID=%v", ids)
	}
}

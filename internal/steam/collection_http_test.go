package steam_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	"github.com/shigaresso/player-lens/internal/steam"
)

func TestCollectReviewsWithHTTPClient(t *testing.T) {
	var mu sync.Mutex
	var requests []steam.PageOptions
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		var input struct {
			AppID     steam.AppID `json:"appid"`
			Cursor    string      `json:"cursor"`
			PageSize  int         `json:"num_per_page"`
			Languages []string    `json:"languages"`
		}
		if err := json.Unmarshal([]byte(r.URL.Query().Get("input_json")), &input); err != nil {
			t.Error(err)
			http.Error(w, "invalid request", 400)
			return
		}
		if input.AppID != 440 || !reflect.DeepEqual(input.Languages, []string{"english"}) {
			t.Errorf("input=%#v", input)
		}
		requests = append(requests, steam.PageOptions{Language: "english", Cursor: input.Cursor, PageSize: input.PageSize})
		response := `{"response":{"reviews":[{"recommendationid":"1"},{"recommendationid":"2"}],"cursor":"A+/="}}`
		if len(requests) == 2 {
			response = `{"response":{"reviews":[{"recommendationid":"3"}],"cursor":"B"}}`
		}
		if _, err := fmt.Fprint(w, response); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	client, err := steam.NewClient(server.Client(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	got, err := steam.CollectReviews(context.Background(), client, 440, steam.CollectionOptions{PageSize: 2, MaxReviews: 3})
	mu.Lock()
	defer mu.Unlock()
	wantRequests := []steam.PageOptions{{Language: "english", Cursor: "*", PageSize: 2}, {Language: "english", Cursor: "A+/=", PageSize: 1}}
	if err != nil || got.StopReason != steam.StopReviewLimit || got.PagesFetched != 2 || len(got.Reviews) != 3 || got.Reviews[2].RecommendationID != 3 || !reflect.DeepEqual(requests, wantRequests) {
		t.Fatalf("result=%#v requests=%#v error=%v", got, requests, err)
	}
}

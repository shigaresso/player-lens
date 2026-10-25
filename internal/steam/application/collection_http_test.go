package application_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	steamadapter "github.com/shigaresso/player-lens/internal/steam/adapter"
	steamapplication "github.com/shigaresso/player-lens/internal/steam/application"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestCollectReviewsWithHTTPClient(t *testing.T) {
	var mu sync.Mutex
	var requests []steamdomain.PageOptions
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		var input struct {
			AppID     steamdomain.AppID `json:"appid"`
			Cursor    string            `json:"cursor"`
			PageSize  int               `json:"num_per_page"`
			Languages []string          `json:"languages"`
		}
		if err := json.Unmarshal([]byte(r.URL.Query().Get("input_json")), &input); err != nil {
			t.Error(err)
			http.Error(w, "invalid request", 400)
			return
		}

		if input.AppID != 440 || !reflect.DeepEqual(input.Languages, []string{"english"}) {
			t.Errorf("input=%#v", input)
		}

		requests = append(requests, steamdomain.PageOptions{Language: "english", Cursor: input.Cursor, PageSize: input.PageSize})
		response := `{"response":{"reviews":[{"recommendationid":"1"},{"recommendationid":"2"}],"cursor":"A+/="}}`
		if len(requests) == 2 {
			response = `{"response":{"reviews":[{"recommendationid":"3"}],"cursor":"B"}}`
		}

		if _, err := fmt.Fprint(w, response); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	client, err := steamadapter.NewClient(server.Client(), server.URL)
	if err != nil {
		t.Fatal(err)
	}

	got, err := steamapplication.CollectReviews(context.Background(), client, 440, collectionOptions(t, steamdomain.CollectionOptionsConfig{PageSize: 2, MaxReviews: 3}))
	mu.Lock()
	defer mu.Unlock()

	wantRequests := []steamdomain.PageOptions{{Language: "english", Cursor: "*", PageSize: 2}, {Language: "english", Cursor: "A+/=", PageSize: 1}}
	if err != nil || got.StopReason != steamdomain.StopReviewLimit || got.PagesFetched != 2 || len(got.Reviews) != 3 || got.Reviews[2].RecommendationID != 3 || !reflect.DeepEqual(requests, wantRequests) {
		t.Fatalf("result=%#v requests=%#v error=%v", got, requests, err)
	}
}

func TestCollectReviewsNumericRecommendationIDs(t *testing.T) {
	responses := []string{
		`{"response":{"reviews":[{"recommendationid":"0000","review":"最初のゼロ"},{"recommendationid":"0001","review":"最初の1"}],"cursor":"A"}}`,
		`{"response":{"reviews":[{"recommendationid":"0","review":"後のゼロ"},{"recommendationid":"1","review":"後の1"}],"cursor":"B"}}`,
		`{"response":{"reviews":[{"recommendationid":"18446744073709551615","review":"最大ID"}],"cursor":"C"}}`,
	}
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		i := int(calls.Add(1)) - 1
		if i >= len(responses) {
			t.Error("予定したページ数を超えて取得した")
			http.Error(w, "unexpected page", http.StatusInternalServerError)
			return
		}

		if _, err := fmt.Fprint(w, responses[i]); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	client, err := steamadapter.NewClient(server.Client(), server.URL)
	if err != nil {
		t.Fatal(err)
	}

	got, err := steamapplication.CollectReviews(t.Context(), client, 440, collectionOptions(t, steamdomain.CollectionOptionsConfig{PageSize: 2, MaxReviews: 4, MaxPages: 3}))
	want := []steamdomain.Review{
		{RecommendationID: 0, Text: "最初のゼロ"},
		{RecommendationID: 1, Text: "最初の1"},
		{RecommendationID: 18446744073709551615, Text: "最大ID"},
	}
	if err != nil || calls.Load() != 3 || got.PagesFetched != 3 || got.StopReason != steamdomain.StopPageLimit || !reflect.DeepEqual(got.Reviews, want) {
		t.Fatalf("result=%#v calls=%d error=%v", got, calls.Load(), err)
	}
}

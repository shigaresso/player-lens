package adapter_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	steamadapter "github.com/shigaresso/player-lens/internal/steam/adapter"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestFetchReviewsPageHTTPStatus(t *testing.T) {
	for _, status := range []int{http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusFound, http.StatusNoContent} {
		t.Run(fmt.Sprintf("HTTP%dを再試行せず拒否する", status), func(t *testing.T) {
			calls, redirects, callbacks := 0, 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/redirect" {
					redirects++
					if _, err := fmt.Fprint(w, `{"response":{}}`); err != nil {
						t.Error(err)
					}

					return
				}

				calls++
				w.Header().Set("Location", "/redirect")
				w.WriteHeader(status)
			}))
			t.Cleanup(server.Close)
			httpClient := server.Client()
			httpClient.CheckRedirect = func(*http.Request, []*http.Request) error { callbacks++; return nil }
			client, err := steamadapter.NewClient(httpClient, server.URL)
			if err != nil {
				t.Fatal(err)
			}

			page, err := client.FetchReviewsPage(context.Background(), 440, steamdomain.PageOptions{})
			var statusError *steamadapter.HTTPStatusError
			if !errors.As(err, &statusError) || statusError.StatusCode != status || !reflect.DeepEqual(page, steamdomain.ReviewPage{}) || calls != 1 || redirects != 0 || callbacks != 0 {
				t.Fatalf("page=%#v error=%v calls=%d redirects=%d callbacks=%d", page, err, calls, redirects, callbacks)
			}

			if err := httpClient.CheckRedirect(nil, nil); err != nil || callbacks != 1 {
				t.Fatal("呼び出し側のクライアントを変更した")
			}
		})
	}
}

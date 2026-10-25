package adapter_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	steamadapter "github.com/shigaresso/player-lens/internal/steam/adapter"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

type panickingReader struct{ cause error }

func (r panickingReader) Read([]byte) (int, error) { panic(r.cause) }

func TestFetchReviewsPageBodyPanic(t *testing.T) {
	cause := errors.New("body read panic")
	var requestContext context.Context
	body := &trackedBody{reader: panickingReader{cause}}
	body.onClose = func() {
		if requestContext == nil || requestContext.Err() != nil {
			t.Error("panic の後始末で Body より先に取得用 context を解放した")
		}
	}
	client, err := steamadapter.NewClient(&http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requestContext = request.Context()
		return &http.Response{StatusCode: http.StatusOK, Body: body, Header: make(http.Header)}, nil
	})}, "")
	if err != nil {
		t.Fatal(err)
	}

	func() {
		defer func() {
			if got := recover(); got != cause {
				t.Errorf("panic の原因=%v; want=%v", got, cause)
			}
		}()

		_, err := client.FetchReviewsPage(t.Context(), 440, steamdomain.PageOptions{})
		t.Errorf("panic を呼び出し側へ伝えなかった: err=%v", err)
	}()
	if body.closed != 1 {
		t.Fatalf("panic 後の Close 回数=%d; want=1", body.closed)
	}

	if requestContext == nil || !errors.Is(requestContext.Err(), context.Canceled) || t.Context().Err() != nil {
		t.Fatal("panic 後に所有する context だけを解放しなかった")
	}
}

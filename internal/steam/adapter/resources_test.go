package adapter_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	steamadapter "github.com/shigaresso/player-lens/internal/steam/adapter"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

type trackedBody struct {
	reader            io.Reader
	readErr, closeErr error
	closed, bytesRead int
	onClose           func()
}

func (b *trackedBody) Read(p []byte) (int, error) {
	if b.readErr != nil {
		return 0, b.readErr
	}
	n, err := b.reader.Read(p)
	b.bytesRead += n
	return n, err
}
func (b *trackedBody) Close() error {
	b.closed++
	if b.onClose != nil {
		b.onClose()
	}
	return b.closeErr
}

func TestFetchReviewsPageBodyLifecycle(t *testing.T) {
	readFailure, closeFailure := errors.New("read failure"), errors.New("close failure")
	for _, tt := range []struct {
		name, content              string
		status                     int
		readErr, closeErr, wantErr error
		wantStatus                 int
	}{
		{"成功時にBodyを閉じる", `{"response":{}}`, 200, nil, nil, nil, 0},
		{"HTTP失敗時に読まず閉じる", "", 500, nil, nil, nil, 500},
		{"JSON失敗時に閉じる", `{`, 200, nil, nil, steamadapter.ErrInvalidResponse, 0},
		{"サイズ超過でも読込を打ち切って閉じる", `{"response":{}}` + strings.Repeat(" ", 3*1024*1024), 200, nil, nil, steamadapter.ErrResponseTooLarge, 0},
		{"読み取り失敗を保持して閉じる", "", 200, readFailure, nil, readFailure, 0},
		{"Close失敗では成功したページを返さない", `{"response":{}}`, 200, nil, closeFailure, closeFailure, 0},
		{"読み取りとCloseの両原因を保持する", "", 200, readFailure, closeFailure, readFailure, 0},
		{"HTTP失敗とCloseの両原因を保持する", "", 429, nil, closeFailure, closeFailure, 429},
		{"JSON解釈とCloseの両原因を保持する", `{`, 200, nil, closeFailure, steamadapter.ErrInvalidResponse, 0},
		{"サイズ超過とCloseの両原因を保持する", `{"response":{}}` + strings.Repeat(" ", 3*1024*1024), 200, nil, closeFailure, steamadapter.ErrResponseTooLarge, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			body := &trackedBody{reader: strings.NewReader(tt.content), readErr: tt.readErr, closeErr: tt.closeErr}
			var requestContext context.Context
			body.onClose = func() {
				if requestContext == nil || requestContext.Err() != nil {
					t.Error("Body を閉じる前に取得用 context を解放した")
				}
			}
			client, err := steamadapter.NewClient(&http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				requestContext = request.Context()
				return &http.Response{StatusCode: tt.status, Body: body, Header: make(http.Header)}, nil
			})}, "")
			if err != nil {
				t.Fatal(err)
			}
			page, err := client.FetchReviewsPage(context.Background(), 440, steamdomain.PageOptions{})
			if requestContext == nil || !errors.Is(requestContext.Err(), context.Canceled) {
				t.Error("操作終了後に取得用 context を解放しなかった")
			}
			if body.closed != 1 || body.bytesRead > 2*1024*1024+1 || (tt.status != 200 && body.bytesRead != 0) {
				t.Fatalf("Body closed=%d bytes=%d", body.closed, body.bytesRead)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("error=%v want=%v", err, tt.wantErr)
			}
			if tt.closeErr != nil && !errors.Is(err, tt.closeErr) {
				t.Errorf("Closeの原因を失った: %v", err)
			}
			if tt.wantStatus != 0 {
				var statusError *steamadapter.HTTPStatusError
				if !errors.As(err, &statusError) || statusError.StatusCode != tt.wantStatus {
					t.Errorf("HTTPの原因を失った: %v", err)
				}
			}
			if tt.wantErr != nil || tt.wantStatus != 0 {
				if !reflect.DeepEqual(page, steamdomain.ReviewPage{}) {
					t.Errorf("部分結果を返した: %#v", page)
				}
			} else if err != nil || page.AppID != 440 {
				t.Errorf("page=%#v error=%v", page, err)
			}
		})
	}
}

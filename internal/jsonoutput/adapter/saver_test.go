package adapter_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	jsonoutput "github.com/shigaresso/player-lens/internal/jsonoutput/adapter"
	steamapplication "github.com/shigaresso/player-lens/internal/steam/application"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

type testWriter struct {
	bytes.Buffer
	calls  int
	closed bool
	short  bool
	err    error
}

func (w *testWriter) Write(data []byte) (int, error) {
	w.calls++
	if w.err != nil {
		return 0, w.err
	}
	if w.short {
		return len(data) - 1, nil
	}
	return w.Buffer.Write(data)
}
func (w *testWriter) Close() error { w.closed = true; return nil }

func TestSaveCollection(t *testing.T) {
	for _, test := range []struct {
		name     string
		short    bool
		writeErr error
		canceled bool
		expired  bool
		wantErr  error
		calls    int
	}{
		{"JSONと改行を一度書き込みCloseしない", false, nil, false, false, nil, 1},
		{"書き込み失敗の原因を保持し再試行しない", false, io.ErrClosedPipe, false, false, io.ErrClosedPipe, 1},
		{"短い書き込みを成功にしない", true, nil, false, false, io.ErrShortWrite, 1},
		{"キャンセル後は書き込まない", false, nil, true, false, context.Canceled, 0},
		{"期限切れ後は書き込まない", false, nil, false, true, context.DeadlineExceeded, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			writer := &testWriter{short: test.short, err: test.writeErr}
			saver, err := jsonoutput.NewSaver(writer)
			if err != nil {
				t.Fatal(err)
			}
			var destination steamapplication.CollectionSaver = saver
			ctx, cancel := context.WithCancel(t.Context())
			t.Cleanup(cancel)
			if test.canceled {
				cancel()
			}
			if test.expired {
				var stop context.CancelFunc
				ctx, stop = context.WithDeadline(ctx, time.Now().Add(-time.Second))
				t.Cleanup(stop)
			}
			input := steamdomain.Collection{AppID: 440, Reviews: []steamdomain.Review{}, StopReason: steamdomain.StopExhausted}
			err = destination.SaveCollection(ctx, input)
			if !errors.Is(err, test.wantErr) || writer.calls != test.calls || writer.closed {
				t.Fatalf("err=%v calls=%d closed=%v", err, writer.calls, writer.closed)
			}
			if test.wantErr == nil {
				want, err := jsonoutput.MarshalCollection(input)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(writer.Bytes(), append(want, '\n')) {
					t.Fatalf("JSONと改行: %q", writer.Bytes())
				}
			}
		})
	}
}

func TestNewSaverRejectsNilWriter(t *testing.T) {
	saver, err := jsonoutput.NewSaver(nil)
	if saver != nil || err == nil {
		t.Fatalf("saver=%v err=%v", saver, err)
	}
}

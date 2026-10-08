package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/shigaresso/player-lens/internal/jsonoutput"
	"github.com/shigaresso/player-lens/internal/steam"
)

type fetcherFunc func(context.Context, uint32, steam.PageOptions) (steam.ReviewPage, error)

func (f fetcherFunc) FetchReviewsPage(ctx context.Context, id uint32, options steam.PageOptions) (steam.ReviewPage, error) {
	return f(ctx, id, options)
}

type failedWriter struct{}

func (failedWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestRunOutputAndExit(t *testing.T) {
	for _, test := range []struct {
		name       string
		args       []string
		fail       bool
		cancel     bool
		outputFail bool
		code       int
		calls      int
		wantJSON   bool
		reason     steam.StopReason
	}{
		{"成功はJSONだけをstdoutへ出す", []string{"--app-id=440", "--max-reviews=1"}, false, false, false, 0, 1, true, steam.StopReviewLimit},
		{"helpはstderrへ出しI/Oしない", []string{"--help"}, false, false, false, 0, 0, false, ""},
		{"不正入力は終了コード2", []string{"--app-id=0"}, false, false, false, 2, 0, false, ""},
		{"取得失敗の部分結果はJSONと終了コード1", []string{"--app-id=440"}, true, false, false, 1, 2, true, steam.StopFailed},
		{"キャンセル後も部分結果を出力する", []string{"--app-id=440"}, false, true, false, 1, 1, true, steam.StopFailed},
		{"出力失敗は終了コード1", []string{"--app-id=440", "--max-reviews=1"}, false, false, true, 1, 1, false, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			t.Cleanup(cancel)
			var stdout, stderr bytes.Buffer
			var writer io.Writer = &stdout
			if test.outputFail {
				writer = failedWriter{}
			}
			saver, err := jsonoutput.NewSaver(writer)
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			fetcher := fetcherFunc(func(context.Context, uint32, steam.PageOptions) (steam.ReviewPage, error) {
				calls++
				if calls == 2 && test.fail {
					return steam.ReviewPage{}, io.ErrUnexpectedEOF
				}
				if test.cancel {
					cancel()
				}
				return steam.ReviewPage{Reviews: []steam.Review{{ID: "1", Text: "本文"}}, NextCursor: "next"}, nil
			})
			code := run(ctx, test.args, fetcher, saver, &stderr)
			if code != test.code || calls != test.calls {
				t.Fatalf("code=%d calls=%d stderr=%q", code, calls, stderr.String())
			}
			if test.wantJSON {
				var result struct {
					Reviews []struct {
						ID string `json:"id"`
					} `json:"reviews"`
					Reason steam.StopReason `json:"stop_reason"`
				}
				if err := json.Unmarshal(stdout.Bytes(), &result); err != nil || len(result.Reviews) != 1 || result.Reviews[0].ID != "1" || result.Reason != test.reason {
					t.Fatalf("JSON=%q err=%v", stdout.String(), err)
				}
			} else if stdout.Len() != 0 {
				t.Fatalf("JSON以外をstdoutへ出した: %q", stdout.String())
			}
			if test.code == 0 && test.calls > 0 && stderr.Len() != 0 {
				t.Fatalf("成功時の診断: %q", stderr.String())
			}
			if test.code != 0 && stderr.Len() == 0 {
				t.Fatal("失敗の診断がない")
			}
			if test.calls == 0 && test.code == 0 && !strings.Contains(stderr.String(), "使用方法") {
				t.Fatal("helpがない")
			}
		})
	}
}

func TestRunDiagnosticFailure(t *testing.T) {
	if code := run(t.Context(), []string{"--help"}, nil, nil, failedWriter{}); code != 1 {
		t.Fatalf("診断出力の失敗: %d", code)
	}
}

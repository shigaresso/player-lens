package page

import (
	"errors"
	"io"
	"net/http/httptest"
	"testing"
)

func TestServePageResponse(t *testing.T) {
	for _, tc := range []struct {
		name, method, header, value string
		status                      int
	}{
		{"GETの本文を返す", "GET", "", "", 200},
		{"HEADは本文なしで返す", "HEAD", "", "", 200},
		{"範囲指定の部分応答を返す", "GET", "Range", "bytes=1-3", 206},
		{"複数範囲の応答を維持する", "GET", "Range", "bytes=0-1,4-5", 206},
		{"範囲外の要求を拒否する", "GET", "Range", "bytes=100-", 416},
		{"条件付き要求で未変更を返す", "GET", "If-None-Match", "*", 304},
		{"不一致の前提条件を拒否する", "GET", "If-Match", "\"missing\"", 412},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(tc.method, "/", nil)
			if tc.header != "" {
				request.Header.Set(tc.header, tc.value)
			}

			writer := httptest.NewRecorder()
			writer.Header().Set("Content-Security-Policy", "default-src 'none'")
			result := Serve(writer, request, []byte("abcdef"))
			if result.StatusCode != tc.status || writer.Code != tc.status || writer.Header().Get("Content-Security-Policy") != "default-src 'none'" {
				t.Fatalf("result=%+v status=%d headers=%v", result, writer.Code, writer.Header())
			}

			if tc.method == "HEAD" && writer.Body.Len() != 0 {
				t.Fatal("HEAD に本文を返した")
			}

			if tc.header == "" && tc.method == "GET" && writer.Body.String() != "abcdef" {
				t.Fatal("GET の本文を変えた")
			}

			if tc.value == "bytes=1-3" && writer.Body.String() != "bcd" {
				t.Fatal("部分応答の本文を変えた")
			}
		})
	}
}

type failingPageWriter struct {
	*httptest.ResponseRecorder
	cause error
	short bool
	calls int
}

func (w *failingPageWriter) Write(data []byte) (int, error) {
	w.calls++
	if w.short {
		return len(data) - 1, nil
	}

	return 0, w.cause
}

func TestServePageWriteFailure(t *testing.T) {
	cause := errors.New("private write diagnostic")
	for _, tc := range []struct {
		name, ranges string
		status       int
		short        bool
	}{
		{"通常応答の失敗を保持して再送しない", "", 200, false},
		{"部分応答の失敗でも206を維持する", "bytes=1-3", 206, false},
		{"複数範囲の失敗でも再送しない", "bytes=0-1,4-5", 206, false},
		{"拒否本文の失敗でも416を維持する", "bytes=100-", 416, false},
		{"短い書き込みを失敗として観測する", "", 200, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest("GET", "/", nil)
			if tc.ranges != "" {
				request.Header.Set("Range", tc.ranges)
			}

			writer := &failingPageWriter{ResponseRecorder: httptest.NewRecorder(), cause: cause, short: tc.short}
			result := Serve(writer, request, []byte("abcdef"))
			want := cause
			if tc.short {
				want = io.ErrShortWrite
			}

			if !errors.Is(result.WriteErr, want) || result.StatusCode != tc.status || writer.Code != tc.status || writer.calls != 1 {
				t.Fatalf("result=%+v status=%d writes=%d", result, writer.Code, writer.calls)
			}
		})
	}
}

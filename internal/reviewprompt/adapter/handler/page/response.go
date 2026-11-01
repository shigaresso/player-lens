package page

import (
	"bytes"
	"io"
	"net/http"
	"time"
)

// Response はページの HTTP 応答処理が選択したステータスと本文書き込み失敗を表す。
// クライアントの受信・表示の成功は表さない。
type Response struct {
	StatusCode int
	WriteErr   error
}

type responseWriter struct {
	http.ResponseWriter
	result Response
}

// Serve は標準 ServeContent のページ応答を実行し、実際のステータスと本文書き込み失敗を返す。
// 呼び出し側が Content-Type・CSP などのヘッダーと経路を設定する。
func Serve(w http.ResponseWriter, r *http.Request, content []byte) Response {
	writer := responseWriter{ResponseWriter: w, result: Response{StatusCode: http.StatusOK}}
	http.ServeContent(&writer, r, "index.html", time.Time{}, bytes.NewReader(content))
	return writer.result
}

func (w *responseWriter) WriteHeader(status int) {
	w.result.StatusCode = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseWriter) Write(data []byte) (int, error) {
	n, err := w.ResponseWriter.Write(data)
	if w.result.WriteErr == nil {
		w.result.WriteErr = err
		if err == nil && n < len(data) {
			w.result.WriteErr = io.ErrShortWrite
		}
	}

	return n, err
}

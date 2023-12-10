package main

import (
	"bytes"
	"io"
	"net/http"
)

// from: https://github.com/syumai/workers/blob/main/_examples/cache/main.go

type responseWriter struct {
	http.ResponseWriter
	StatusCode int
	Body       []byte
}

func (rw responseWriter) WriteHeader(statusCode int) {
	rw.StatusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw responseWriter) Write(data []byte) (int, error) {
	rw.Body = append(rw.Body, data...)
	return rw.ResponseWriter.Write(data)
}

func (rw *responseWriter) ToHTTPResponse() *http.Response {
	return &http.Response{
		StatusCode: rw.StatusCode,
		Header:     rw.Header(),
		Body:       io.NopCloser(bytes.NewReader(rw.Body)),
	}
}

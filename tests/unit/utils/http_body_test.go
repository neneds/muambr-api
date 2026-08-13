package utils

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"testing"

	"muambr-api/utils"
)

func TestReadDecompressedBody_gzip(t *testing.T) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write([]byte("hello gzip")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	resp := &http.Response{
		Header: http.Header{"Content-Encoding": []string{"gzip"}},
		Body:   io.NopCloser(bytes.NewReader(buf.Bytes())),
	}
	got, err := utils.ReadDecompressedBody(resp)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello gzip" {
		t.Errorf("got %q", got)
	}
}

func TestReadDecompressedBody_identity(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{},
		Body:   io.NopCloser(bytes.NewReader([]byte("plain"))),
	}
	got, err := utils.ReadDecompressedBody(resp)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "plain" {
		t.Errorf("got %q", got)
	}
}

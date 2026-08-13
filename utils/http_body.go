package utils

import (
	"compress/gzip"
	"io"
	"net/http"

	"github.com/dsnet/compress/brotli"
)

// ReadDecompressedBody reads an HTTP response body, decompressing gzip or brotli
// when Content-Encoding is set. Callers that set Accept-Encoding themselves must
// use this — net/http will not auto-decode in that case.
func ReadDecompressedBody(resp *http.Response) ([]byte, error) {
	var reader io.Reader = resp.Body
	contentEncoding := resp.Header.Get("Content-Encoding")

	switch contentEncoding {
	case "gzip":
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			Warn("⚠️ Failed to create gzip reader, falling back to raw read", Error(err))
			return io.ReadAll(resp.Body)
		}
		defer gzipReader.Close()
		reader = gzipReader
	case "br":
		brotliReader, err := brotli.NewReader(resp.Body, nil)
		if err != nil {
			Warn("⚠️ Failed to create brotli reader, falling back to raw read", Error(err))
			return io.ReadAll(resp.Body)
		}
		defer brotliReader.Close()
		reader = brotliReader
	}

	return io.ReadAll(reader)
}

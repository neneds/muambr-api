package utils

import (
	"crypto/tls"
	"net/http"
	"time"
)

// DefaultUserAgent is a current desktop Chrome string. Client hint headers below match it.
const DefaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

var sharedTransport = &http.Transport{
	TLSClientConfig: &tls.Config{
		MinVersion: tls.VersionTLS12,
	},
	MaxIdleConns:        32,
	MaxIdleConnsPerHost: 8,
	IdleConnTimeout:     90 * time.Second,
	TLSHandshakeTimeout: 10 * time.Second,
}

// CreateAntiBotClient returns an HTTP client that reuses the package transport.
// Callers may set Timeout on the returned client without affecting others.
func CreateAntiBotClient() *http.Client {
	return &http.Client{
		Transport: sharedTransport,
		Timeout:   30 * time.Second,
	}
}

// ApplyAntiBotHeaders sets a consistent desktop-Chrome header set.
// Do not set Accept-Encoding: net/http then gzip-decodes automatically.
func ApplyAntiBotHeaders(req *http.Request) {
	req.Header.Set("User-Agent", DefaultUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,application/json;q=0.8,*/*;q=0.7")
	req.Header.Set("Accept-Language", "pt-BR,pt;q=0.9,en;q=0.8")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Sec-Ch-Ua", `"Chromium";v="124", "Google Chrome";v="124", "Not-A.Brand";v="99"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"macOS"`)
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
}

// MakeAntiBotRequest GETs url with the shared client and browser headers.
func MakeAntiBotRequest(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	ApplyAntiBotHeaders(req)
	return CreateAntiBotClient().Do(req)
}

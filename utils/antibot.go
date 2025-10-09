package utils

import (
	"crypto/tls"
	"math/rand"
	"net/http"
	"time"
)

// AntiBotConfig holds configuration for anti-bot strategies
type AntiBotConfig struct {
	UserAgentRotation bool
	RandomDelay       bool
	MinDelay          time.Duration
	MaxDelay          time.Duration
	UseReferer        bool
	RefererURL        string
}

// DefaultAntiBotConfig returns a default configuration for anti-bot protection
func DefaultAntiBotConfig(siteURL string) *AntiBotConfig {
	return &AntiBotConfig{
		UserAgentRotation: true,
		RandomDelay:       true,
		MinDelay:          500 * time.Millisecond,
		MaxDelay:          2000 * time.Millisecond,
		UseReferer:        true,
		RefererURL:        siteURL,
	}
}

// CreateAntiBotClient creates an HTTP client with anti-bot protection measures
func CreateAntiBotClient() *http.Client {
	// Create transport with realistic TLS settings
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			MaxVersion: tls.VersionTLS13,
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &http.Client{
		Transport: tr,
		Timeout:   30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Allow up to 10 redirects
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			// Copy headers to redirected request
			if len(via) > 0 {
				for key, values := range via[0].Header {
					for _, value := range values {
						req.Header.Add(key, value)
					}
				}
			}
			return nil
		},
	}
}

// GetRandomUserAgent returns a random user agent string from a pool of realistic options
func GetRandomUserAgent() string {
	userAgents := []string{
		// Chrome on Windows
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
		
		// Chrome on macOS
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		
		// Safari on macOS
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Safari/605.1.15",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Safari/605.1.15",
		
		// Firefox on Windows
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:120.0) Gecko/20100101 Firefox/120.0",
		
		// Edge on Windows
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36 Edg/121.0.0.0",
	}
	
	return userAgents[rand.Intn(len(userAgents))]
}

// ApplyAntiBotHeaders applies comprehensive anti-bot headers to an HTTP request
func ApplyAntiBotHeaders(req *http.Request, config *AntiBotConfig) {
	// Set User-Agent
	if config.UserAgentRotation {
		req.Header.Set("User-Agent", GetRandomUserAgent())
	} else {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36")
	}

	// Core browser headers
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Accept-Language", "pt-BR,pt;q=0.9,en;q=0.8,en-US;q=0.7")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
	
	// Modern Chrome security headers
	req.Header.Set("Sec-Ch-Ua", `"Not_A Brand";v="8", "Chromium";v="120", "Google Chrome";v="120"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"macOS"`)
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	
	// Connection headers
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Connection", "keep-alive")
	
	// Add referer if configured
	if config.UseReferer && config.RefererURL != "" {
		req.Header.Set("Referer", config.RefererURL)
	}
	
	// Randomly add some optional headers to look more natural
	if rand.Float32() < 0.3 {
		req.Header.Set("DNT", "1")
	}
	
	if rand.Float32() < 0.2 {
		req.Header.Set("Sec-GPC", "1")
	}
}

// RandomDelay introduces a random delay to simulate human behavior
func RandomDelay(config *AntiBotConfig) {
	if !config.RandomDelay {
		return
	}
	
	minMs := int(config.MinDelay.Milliseconds())
	maxMs := int(config.MaxDelay.Milliseconds())
	
	if maxMs <= minMs {
		return
	}
	
	delay := time.Duration(rand.Intn(maxMs-minMs)+minMs) * time.Millisecond
	time.Sleep(delay)
}

// MakeAntiBotRequest performs an HTTP request with anti-bot protection measures
func MakeAntiBotRequest(url string, config *AntiBotConfig) (*http.Response, error) {
	// Apply random delay before making request
	RandomDelay(config)
	
	// Create anti-bot client
	client := CreateAntiBotClient()
	
	// Create request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	// Apply anti-bot headers
	ApplyAntiBotHeaders(req, config)
	
	// Make the request
	return client.Do(req)
}

// init initializes the random seed
func init() {
	rand.Seed(time.Now().UnixNano())
}
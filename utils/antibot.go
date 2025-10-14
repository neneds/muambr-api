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
		MinDelay:          800 * time.Millisecond,  // Increased delay to look more human
		MaxDelay:          3000 * time.Millisecond, // Increased max delay
		UseReferer:        true,
		RefererURL:        "https://www.google.com/", // Use Google as referer
	}
}

// CreateAntiBotClient creates an HTTP client with anti-bot protection measures
func CreateAntiBotClient() *http.Client {
	// Create transport with realistic TLS settings
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion:               tls.VersionTLS12,
			MaxVersion:               tls.VersionTLS13,
			PreferServerCipherSuites: false,
			InsecureSkipVerify:       false,
		},
		ForceAttemptHTTP2:     false, // Disable HTTP/2 to avoid connection header conflicts
		MaxIdleConns:          10,    // Reduce to look less bot-like
		MaxIdleConnsPerHost:   2,     // Limit connections per host
		IdleConnTimeout:       30 * time.Second, // Reduce timeout
		TLSHandshakeTimeout:   15 * time.Second, // Increase for more realistic timing
		ExpectContinueTimeout: 2 * time.Second,  // Slightly increase
		DisableKeepAlives:     false,            // Keep alive for realism
		DisableCompression:    false,            // Enable compression
		MaxConnsPerHost:       5,                // Limit concurrent connections
		ResponseHeaderTimeout: 30 * time.Second, // Add response timeout
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
		// Latest Chrome on Windows (most common)
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36",
		
		// Latest Chrome on macOS
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36",
		
		// Latest Safari on macOS
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Safari/605.1.15",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.6 Safari/605.1.15",
		
		// Latest Firefox
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:130.0) Gecko/20100101 Firefox/130.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:130.0) Gecko/20100101 Firefox/130.0",
		
		// Latest Edge
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36 Edg/129.0.0.0",
		
		// Mobile user agents for better disguise
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.6 Mobile/15E148 Safari/604.1",
		"Mozilla/5.0 (Linux; Android 14; SM-G998B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Mobile Safari/537.36",
	}
	
	return userAgents[rand.Intn(len(userAgents))]
}

// ApplyAntiBotHeaders applies comprehensive anti-bot headers to an HTTP request
func ApplyAntiBotHeaders(req *http.Request, config *AntiBotConfig) {
	// Set User-Agent
	userAgent := ""
	if config.UserAgentRotation {
		userAgent = GetRandomUserAgent()
	} else {
		userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36"
	}
	req.Header.Set("User-Agent", userAgent)

	// Core browser headers - match what real browsers send
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Accept-Language", "pt-BR,pt;q=0.9,en;q=0.8,en-US;q=0.7")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	
	// Remove cache headers that look suspicious
	// req.Header.Set("Cache-Control", "no-cache")
	// req.Header.Set("Pragma", "no-cache")
	
	// Modern Chrome security headers - update to latest versions
	req.Header.Set("Sec-Ch-Ua", `"Google Chrome";v="129", "Not=A?Brand";v="8", "Chromium";v="129"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"Windows"`)
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	
	// Connection headers
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	// Note: Don't set Connection header as it can conflict with HTTP/2
	
	// Add more realistic headers
	req.Header.Set("Sec-Purpose", "prefetch;prerender")
	
	// Add referer if configured - but make it more realistic
	if config.UseReferer && config.RefererURL != "" {
		// Use Google as referer to look more natural
		req.Header.Set("Referer", "https://www.google.com/")
	}
	
	// Randomly add some optional headers to look more natural
	if rand.Float32() < 0.3 {
		req.Header.Set("DNT", "1")
	}
	
	if rand.Float32() < 0.2 {
		req.Header.Set("Sec-GPC", "1")
	}
	
	// Add some randomness to make requests look more human
	if rand.Float32() < 0.5 {
		req.Header.Set("Accept-CH", "Sec-CH-UA, Sec-CH-UA-Arch, Sec-CH-UA-Bitness, Sec-CH-UA-Full-Version, Sec-CH-UA-Full-Version-List, Sec-CH-UA-Mobile, Sec-CH-UA-Model, Sec-CH-UA-Platform, Sec-CH-UA-Platform-Version")
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
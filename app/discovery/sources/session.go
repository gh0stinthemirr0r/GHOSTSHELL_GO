package discovery

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/projectdiscovery/ratelimit"
	"github.com/projectdiscovery/retryablehttp-go"
	errorutil "github.com/projectdiscovery/utils/errors"
)

// Session handles agent sessions with rate-limiting and retries
type Session struct {
	Client     *retryablehttp.Client
	Keys       *Keys
	RateLimits *ratelimit.MultiLimiter
	RetryMax   int
}

// NewSession creates a new session with the specified parameters
func NewSession(keys *Keys, retryMax, timeout, rateLimit int, engines []string, duration time.Duration) (*Session, error) {
	// Configure HTTP transport
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		ResponseHeaderTimeout: time.Duration(timeout) * time.Second,
		Proxy:                 http.ProxyFromEnvironment,
	}

	client := retryablehttp.NewWithHTTPClient(&http.Client{
		Transport: transport,
		Timeout:   time.Duration(timeout) * time.Second,
	}, retryablehttp.Options{RetryMax: retryMax})

	// Initialize session
	session := &Session{
		Client:   client,
		Keys:     keys,
		RetryMax: retryMax,
	}

	// Configure rate limits
	defaultRateLimit := &ratelimit.Options{Key: "default", MaxCount: uint(rateLimit), Duration: duration}
	if rateLimit <= 0 {
		defaultRateLimit.IsUnlimited = true
	}

	ratelimiter, err := ratelimit.NewMultiLimiter(context.Background(), defaultRateLimit)
	if err != nil {
		return nil, err
	}
	session.RateLimits = ratelimiter

	for _, engine := range engines {
		engineRateLimit := DefaultRateLimits[engine]
		if engineRateLimit == nil {
			engineRateLimit = defaultRateLimit
			engineRateLimit.Key = engine
		}
		if err := ratelimiter.Add(engineRateLimit); err != nil {
			return nil, errorutil.NewWithErr(err).Msgf("Failed to add rate limit for engine %s", engine)
		}
	}

	return session, nil
}

// Do executes an HTTP request with rate-limiting
func (s *Session) Do(request *retryablehttp.Request, source string) (*http.Response, error) {
	if err := s.RateLimits.Take(source); err != nil {
		return nil, err
	}
	response, err := s.Client.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		return response, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}
	return response, nil
}

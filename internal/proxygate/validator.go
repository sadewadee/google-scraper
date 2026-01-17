package proxygate

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
)

type Validator struct {
	pool        *Pool
	concurrency int
}

func NewValidator(concurrency int, pool *Pool) *Validator {
	return &Validator{
		pool:        pool,
		concurrency: concurrency,
	}
}

func (v *Validator) Run(ctx context.Context) error {
	egroup, ctx := errgroup.WithContext(ctx)

	for i := 0; i < v.concurrency; i++ {
		egroup.Go(func() error {
			return v.worker(ctx)
		})
	}

	return egroup.Wait()
}

func (v *Validator) worker(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case rawProxy := <-v.pool.raw:
			// Construct proper URL if no scheme present (raw format: IP:PORT)
			proxyURL := rawProxy
			if !strings.Contains(rawProxy, "://") {
				proxyURL = "socks5://" + rawProxy
			}
			if v.validate(ctx, proxyURL) {
				v.pool.AddValidated(rawProxy) // Store original format
			}
		}
	}
}

func (v *Validator) validate(ctx context.Context, proxyURL string) bool {
	// Step 1: Ping Google
	if !v.checkURL(ctx, proxyURL, "https://www.google.com") {
		return false
	}

	// Step 2: Verify Google Maps
	return v.checkURL(ctx, proxyURL, "https://www.google.com/maps")
}

func (v *Validator) checkURL(ctx context.Context, proxyURL, testURL string) bool {
	proxyFunc := http.ProxyURL(mustParseURL(proxyURL))

	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			Proxy: proxyFunc,
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, testURL, nil)
	if err != nil {
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode < 400
}

func mustParseURL(s string) *url.URL {
	u, _ := url.Parse(s)
	return u
}

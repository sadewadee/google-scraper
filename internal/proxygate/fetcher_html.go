package proxygate

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ProxyDBEntry represents a parsed proxy from proxydb.net
type ProxyDBEntry struct {
	IP       string
	Port     int
	Country  string
	Uptime   float64 // percentage
	Response float64 // seconds
}

// FetchProxyDB fetches and parses proxies from proxydb.net
// Returns only high-quality proxies (uptime > minUptime%, response < maxResponse seconds)
func FetchProxyDB(ctx context.Context, minUptime float64, maxResponse float64) ([]string, error) {
	url := "https://proxydb.net/?protocol=socks5&country"

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set headers to look like a browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch proxydb: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("proxydb returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	entries := parseProxyDBHTML(string(body))

	// Filter by quality criteria
	var result []string
	for _, entry := range entries {
		if entry.Uptime >= minUptime && entry.Response <= maxResponse {
			result = append(result, fmt.Sprintf("%s:%d", entry.IP, entry.Port))
		}
	}

	log.Printf("[ProxyGate] Fetched %d proxies from proxydb.net, %d passed quality filter (uptime>=%.0f%%, response<=%.1fs)",
		len(entries), len(result), minUptime, maxResponse)

	return result, nil
}

// parseProxyDBHTML extracts proxy entries from proxydb.net HTML
func parseProxyDBHTML(html string) []ProxyDBEntry {
	var entries []ProxyDBEntry

	// Pattern to match IP:port in table rows
	// ProxyDB format: <td>IP:PORT</td> or similar
	ipPortRegex := regexp.MustCompile(`(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}):(\d{2,5})`)

	// Pattern to match uptime percentage like "93.51%"
	uptimeRegex := regexp.MustCompile(`(\d+\.?\d*)\s*%`)

	// Pattern to match response time like "3.7s"
	responseRegex := regexp.MustCompile(`(\d+\.?\d*)\s*s`)

	// Split by table rows
	rows := strings.Split(html, "<tr")

	for _, row := range rows {
		// Skip header rows
		if strings.Contains(row, "<th") {
			continue
		}

		// Find IP:port
		ipMatch := ipPortRegex.FindStringSubmatch(row)
		if ipMatch == nil {
			continue
		}

		ip := ipMatch[1]
		port, err := strconv.Atoi(ipMatch[2])
		if err != nil {
			continue
		}

		// Find uptime
		var uptime float64
		uptimeMatches := uptimeRegex.FindAllStringSubmatch(row, -1)
		if len(uptimeMatches) > 0 {
			// First percentage is usually uptime
			uptime, _ = strconv.ParseFloat(uptimeMatches[0][1], 64)
		}

		// Find response time
		var response float64
		responseMatches := responseRegex.FindAllStringSubmatch(row, -1)
		if len(responseMatches) > 0 {
			response, _ = strconv.ParseFloat(responseMatches[0][1], 64)
		}

		// Extract country from flag emoji or country code
		country := extractCountry(row)

		entries = append(entries, ProxyDBEntry{
			IP:       ip,
			Port:     port,
			Country:  country,
			Uptime:   uptime,
			Response: response,
		})
	}

	return entries
}

// extractCountry tries to extract country code from row
func extractCountry(row string) string {
	// Look for country codes like (RU), (US), (SG), etc.
	countryRegex := regexp.MustCompile(`\(([A-Z]{2})\)`)
	match := countryRegex.FindStringSubmatch(row)
	if match != nil {
		return match[1]
	}
	return ""
}

// AddProxyDBSource adds proxydb.net as a special source that uses HTML scraping
// This is called by the fetcher when it detects a proxydb.net URL
func (f *Fetcher) fetchProxyDB(ctx context.Context) error {
	// Quality filter: uptime >= 70%, response <= 5 seconds
	proxies, err := FetchProxyDB(ctx, 70.0, 5.0)
	if err != nil {
		return err
	}

	for _, proxy := range proxies {
		select {
		case f.pool.raw <- proxy:
		case <-ctx.Done():
			return nil
		}
	}

	return nil
}

// IsProxyDBURL checks if a URL is proxydb.net
func IsProxyDBURL(url string) bool {
	return strings.Contains(url, "proxydb.net")
}

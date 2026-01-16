package domain

import "time"

// ProxySource represents a source URL for proxies
type ProxySource struct {
	ID        int64     `json:"id"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ProxyStatus represents the status of a proxy
type ProxyStatus string

const (
	ProxyStatusPending ProxyStatus = "pending"
	ProxyStatusHealthy ProxyStatus = "healthy"
	ProxyStatusDead    ProxyStatus = "dead"
	ProxyStatusBanned  ProxyStatus = "banned"
)

// Proxy represents a single proxy server
type Proxy struct {
	ID           int64       `json:"id"`
	IP           string      `json:"ip"`
	Port         int         `json:"port"`
	Protocol     string      `json:"protocol"` // socks5, socks4, http, https
	Country      string      `json:"country,omitempty"`
	Uptime       float64     `json:"uptime,omitempty"`        // percentage 0-100
	ResponseTime float64     `json:"response_time,omitempty"` // seconds
	Status       ProxyStatus `json:"status"`
	LastChecked  *time.Time  `json:"last_checked,omitempty"`
	LastUsed     *time.Time  `json:"last_used,omitempty"`
	FailCount    int         `json:"fail_count"`
	SuccessCount int         `json:"success_count"`
	SourceID     *int64      `json:"source_id,omitempty"`
	SourceURL    string      `json:"source_url,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

// Address returns the proxy address as IP:port
func (p *Proxy) Address() string {
	return p.IP + ":" + string(rune(p.Port))
}

// ProxyListParams contains parameters for listing proxies
type ProxyListParams struct {
	Status   ProxyStatus
	Protocol string
	Country  string
	Limit    int
	Offset   int
}

// ProxyStats contains proxy pool statistics
type ProxyStats struct {
	Total     int `json:"total"`
	Healthy   int `json:"healthy"`
	Dead      int `json:"dead"`
	Banned    int `json:"banned"`
	Pending   int `json:"pending"`
	AvgUptime float64 `json:"avg_uptime"`
}

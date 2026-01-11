package proxygate

import (
	"time"
)

type Config struct {
	Enabled              bool
	ListenAddr           string   // e.g., "localhost:8081"
	SourceURLs           []string // Default GitHub raw URLs
	RefreshInterval      time.Duration
	ValidatorConcurrency int
}

func DefaultConfig() *Config {
	return &Config{
		Enabled:    false,
		ListenAddr: "localhost:8081",
		SourceURLs: []string{
			"https://raw.githubusercontent.com/TheSpeedX/SOCKS-List/master/socks5.txt",
			"https://raw.githubusercontent.com/monosans/proxy-list/main/proxies/socks5.txt",
			"https://raw.githubusercontent.com/hookzof/socks5_list/master/proxy.txt",
			"https://raw.githubusercontent.com/zloi-user/hideip.me/main/socks5.txt",
		},
		RefreshInterval:      10 * time.Minute,
		ValidatorConcurrency: 50,
	}
}

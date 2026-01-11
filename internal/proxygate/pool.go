package proxygate

import (
	"errors"
	"sync"
)

type Pool struct {
	mu      sync.RWMutex
	proxies []string
	index   int
	raw     chan string // Channel for raw fetched proxies
	valid   chan string // Channel for validated proxies
}

func NewPool() *Pool {
	return &Pool{
		proxies: make([]string, 0),
		raw:     make(chan string, 10000),
		valid:   make(chan string, 1000),
	}
}

func (p *Pool) GetNext() (string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.proxies) == 0 {
		return "", errors.New("no healthy proxies available")
	}

	proxy := p.proxies[p.index]
	p.index = (p.index + 1) % len(p.proxies)

	return proxy, nil
}

func (p *Pool) AddValidated(proxy string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Deduplicate
	for _, existing := range p.proxies {
		if existing == proxy {
			return
		}
	}

	p.proxies = append(p.proxies, proxy)
}

func (p *Pool) Remove(proxy string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, existing := range p.proxies {
		if existing == proxy {
			p.proxies = append(p.proxies[:i], p.proxies[i+1:]...)
			break
		}
	}
}

func (p *Pool) Size() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.proxies)
}

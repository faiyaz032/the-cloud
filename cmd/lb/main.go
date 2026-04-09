package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"
)

type LoadBalancer interface {
	GetNextServer() string
}

type RoundRobin struct {
	servers []string
	alive   map[string]bool
	current int
	mu      sync.Mutex
}

func NewRoundRobin(servers []string) *RoundRobin {
	alive := make(map[string]bool, len(servers))
	for _, server := range servers {
		alive[server] = true
	}

	return &RoundRobin{servers: servers, alive: alive}
}

func (rr *RoundRobin) GetNextServer() string {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	if len(rr.servers) == 0 {
		return ""
	}

	next := rr.current
	for i := 0; i < len(rr.servers); i++ {
		server := rr.servers[next]
		next = (next + 1) % len(rr.servers)
		if rr.alive[server] {
			rr.current = next
			return server
		}
	}

	return ""
}

func (rr *RoundRobin) StartHealthChecks(interval, timeout time.Duration) {
	checkServerHealth := func() {
		for _, server := range rr.servers {
			alive := isServerAlive(server, timeout)
			rr.mu.Lock()
			rr.alive[server] = alive
			rr.mu.Unlock()
		}
	}

	checkServerHealth()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		checkServerHealth()
	}
}

func isServerAlive(address string, timeout time.Duration) bool {
	client := http.Client{
		Timeout: timeout,
	}
	resp, err := client.Get("http://" + address)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode < 500
}

const (
	RoundRobinAlgo = "round_robin"
)

func NewLoadBalancer(algo string, servers []string) LoadBalancer {
	switch algo {
	case RoundRobinAlgo:
		return NewRoundRobin(servers)
	default:
		panic("unsupported load balancing algorithm: " + algo)
	}
}

func main() {
	computeLB := NewRoundRobin([]string{
		"192.168.56.11:8080",
		"192.168.56.12:8080",
	})

	storageLB := NewRoundRobin([]string{
		"192.168.57.11:8081",
		"192.168.57.12:8081",
	})

	go computeLB.StartHealthChecks(3*time.Second, 1*time.Second)
	go storageLB.StartHealthChecks(3*time.Second, 1*time.Second)

	http.HandleFunc("/", HandleHTTP(computeLB, storageLB))
	log.Println("HTTP Load Balancer running on :80")
	log.Fatal(http.ListenAndServe(":80", nil))
}

func HandleHTTP(computeLB, storageLB LoadBalancer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var target string

		switch {
		case r.URL.Path == "/" || strings.HasPrefix(r.URL.Path, "/compute"):
			target = computeLB.GetNextServer()
		case strings.HasPrefix(r.URL.Path, "/storage"):
			r.URL.Path = strings.TrimPrefix(r.URL.Path, "/storage")
			target = storageLB.GetNextServer()
		default:
			http.Error(w, "unknown path", http.StatusNotFound)
			return
		}

		if target == "" {
			http.Error(w, "no healthy backend", http.StatusServiceUnavailable)
			return
		}

		remote, err := url.Parse("http://" + target)
		if err != nil {
			http.Error(w, "bad backend URL", http.StatusInternalServerError)
			return
		}

		proxy := httputil.NewSingleHostReverseProxy(remote)
		proxy.ServeHTTP(w, r)
	}
}

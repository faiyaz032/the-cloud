package main

import (
	"io"
	"log"
	"net"
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
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
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
	lb := NewRoundRobin([]string{
		"192.168.56.11:8080", // node01
		"192.168.56.12:8080", // node02
	})

	go lb.StartHealthChecks(3*time.Second, 1*time.Second)

	if err := Start(":80", lb); err != nil {
		log.Fatal(err)
	}
}

func HandleConnection(conn net.Conn, lb LoadBalancer) {
	defer conn.Close()

	serverAddr := lb.GetNextServer()
	if serverAddr == "" {
		log.Println("no healthy backend available")
		return
	}

	serverConn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		log.Println("backend connection error:", err)
		return
	}
	defer serverConn.Close()

	done := make(chan struct{}, 2)

	// client -> backend
	go func() {
		_, _ = io.Copy(serverConn, conn)
		done <- struct{}{}
	}()

	// backend -> client
	go func() {
		_, _ = io.Copy(conn, serverConn)
		done <- struct{}{}
	}()

	<-done
	<-done
}

func Start(address string, lb LoadBalancer) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	defer listener.Close()

	log.Println("listening on", address)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("accept error:", err)
			continue
		}

		go HandleConnection(conn, lb)
	}
}

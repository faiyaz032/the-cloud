package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/faiyaz032/the-cloud/internal/hashing"
)

func main() {
	nodeIP := os.Getenv("NODE_IP")
	home, _ := os.UserHomeDir()
	storagePath := filepath.Join(home, "cloud-data")

	ring := hashing.NewHashRing(20)
	ring.AddNode("192.168.56.13:8081")
	ring.AddNode("192.168.56.14:8081")

	server := NewStorageServer(nodeIP, storagePath, ring)
	http.HandleFunc("/upload", server.HandleUpload)

	log.Printf("Storage node %s starting on :8080...", nodeIP)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

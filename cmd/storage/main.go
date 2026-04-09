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
	if nodeIP == "" {
		log.Fatal("CRITICAL: NODE_IP environment variable is not set")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Could not find home directory: %v", err)
	}

	storagePath := filepath.Join(home, "cloud-data")

	ring := hashing.NewHashRing(20)
	ring.AddNode("192.168.56.13:8081")
	ring.AddNode("192.168.56.14:8081")

	server := NewStorageServer(nodeIP, storagePath, ring)

	http.HandleFunc("/upload", server.HandleUpload)
	http.HandleFunc("/download", server.HandleDownload)

	log.Printf("Storage node %s starting on :8081...", nodeIP)
	log.Printf("Data directory: %s", storagePath)
	log.Fatal(http.ListenAndServe(":8081", nil))
}

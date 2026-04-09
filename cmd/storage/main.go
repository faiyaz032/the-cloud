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

	// Consistent Hash Ring initialization
	ring := hashing.NewHashRing(20)
	ring.AddNode("192.168.57.11")
	ring.AddNode("192.168.57.12")

	server := NewStorageServer(nodeIP, storagePath, ring)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("healthy"))
		if err != nil {
			log.Printf("Error writing response: %v", err)
		}
	})

	http.HandleFunc("/upload", server.HandleUpload)
	http.HandleFunc("/download", server.HandleDownload)

	log.Printf("Storage node %s starting on :8081...", nodeIP)
	log.Printf("Data directory: %s", storagePath)

	// Listening on :8081 allows traffic from any interface (eth0, eth1, eth2)
	log.Fatal(http.ListenAndServe(":8081", nil))
}

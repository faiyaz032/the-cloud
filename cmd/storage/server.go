package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/faiyaz032/the-cloud/internal/hashing"
)

type StorageServer struct {
	ring    *hashing.HashRing
	nodeIP  string
	dataDir string
}

func NewStorageServer(nodeIP, dataDir string, ring *hashing.HashRing) *StorageServer {
	return &StorageServer{
		ring:    ring,
		nodeIP:  nodeIP,
		dataDir: dataDir,
	}
}

func (s *StorageServer) HandleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "Key is required", http.StatusBadRequest)
		return
	}

	isInternal := r.Header.Get("X-Internal-Request") == "true"
	targetNode := s.ring.GetNode(key)

	if targetNode == s.nodeIP || isInternal {
		//save file locally
		err := s.saveLocally(key, r.Body)
		if err != nil {
			http.Error(w, "Failed to save file", http.StatusInternalServerError)
			log.Printf("Error saving file %s: %v", key, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("File saved successfully"))
	} else {
		// forward request to another node
		s.forwardToNode(w, targetNode, key, r)
	}
}

// save files locally into node
func (s *StorageServer) saveLocally(key string, reader io.Reader) error {
	filePath := filepath.Join(s.dataDir, key)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, reader)
	return err
}

func (s *StorageServer) forwardToNode(w http.ResponseWriter, nodeIP string, key string, r *http.Request) {
	// buffer read from request body to forward it
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	r.Body.Close()

	targetURL := fmt.Sprintf("http://%s/upload?key=%s", nodeIP, key)
	proxyReq, _ := http.NewRequest(http.MethodPost, targetURL, io.NopCloser(bytes.NewReader(bodyBytes)))

	// set headers
	proxyReq.Header.Set("X-Internal-Request", "true")
	proxyReq.Header.Set("Content-Type", r.Header.Get("Content-Type"))

	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Forwarding error: %v", err), http.StatusInternalServerError)
		log.Printf("Error forwarding to %s: %v", nodeIP, err)
		return
	}
	defer resp.Body.Close()

	// copy response back to client
	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Printf("Error copying response back: %v", err)
	}
}

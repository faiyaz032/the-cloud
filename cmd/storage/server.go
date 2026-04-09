package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
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

func (s *StorageServer) HandleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract key from URL (e.g., /download?key=my-image.png)
	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "Key is required", http.StatusBadRequest)
		return
	}

	isInternal := r.Header.Get("X-Internal-Request") == "true"
	targetNode := s.ring.GetNode(key)

	// Check if this node is the owner or if it's an internal redirected request
	if targetNode == s.nodeIP || isInternal {
		s.serveLocally(w, r, key)
	} else {
		// Forward the GET request to the correct node
		s.forwardGetToNode(w, targetNode, key)
	}
}

func (s *StorageServer) serveLocally(w http.ResponseWriter, r *http.Request, key string) {
	filePath := filepath.Join(s.dataDir, key)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	log.Printf("Serving file %s from local storage", key)
	http.ServeFile(w, r, filePath)
}

func (s *StorageServer) forwardGetToNode(w http.ResponseWriter, nodeIP string, key string) {
	targetURL := fmt.Sprintf("http://%s:8081/download?key=%s", nodeIP, url.QueryEscape(key))

	proxyReq, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	proxyReq.Header.Set("X-Internal-Request", "true")

	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		http.Error(w, "Error reaching target node", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Printf("Error copying response back: %v", err)
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
	log.Printf("HandleUpload: key=%s, targetNode=%s, nodeIP=%s, isInternal=%v", key, targetNode, s.nodeIP, isInternal)

	if targetNode == s.nodeIP || isInternal {
		//save file locally
		log.Printf("Saving file %s locally on %s", key, s.nodeIP)
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
		log.Printf("Forwarding upload for key %s from %s to %s", key, s.nodeIP, targetNode)
		s.forwardToNode(w, targetNode, key, r)
	}
}

// save files locally into node
func (s *StorageServer) saveLocally(key string, reader io.Reader) error {
	err := os.MkdirAll(s.dataDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	filePath := filepath.Join(s.dataDir, key)
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, reader)
	if err == nil {
		log.Printf("Successfully saved file %s locally", key)
	}
	return err
}

func (s *StorageServer) forwardToNode(w http.ResponseWriter, nodeIP string, key string, r *http.Request) {
	targetURL := fmt.Sprintf("http://%s:8081/upload?key=%s", nodeIP, url.QueryEscape(key))
	proxyReq, err := http.NewRequest(http.MethodPost, targetURL, r.Body)
	if err != nil {
		http.Error(w, "Failed to create forward request", http.StatusInternalServerError)
		log.Printf("Error creating request for %s: %v", nodeIP, err)
		return
	}

	// Reuse original metadata while streaming the request body through.
	proxyReq.Header.Set("X-Internal-Request", "true")
	proxyReq.Header.Set("Content-Type", r.Header.Get("Content-Type"))
	proxyReq.ContentLength = r.ContentLength

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

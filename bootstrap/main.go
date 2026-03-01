package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	version     = "0.1.0"
	peerTimeout = 2 * time.Minute
	port        = 7070
)

type PeerRecord struct {
	ID       string    `json:"id"`
	Address  string    `json:"address"`
	LastSeen time.Time `json:"last_seen"`
}

type BootstrapServer struct {
	peers map[string]*PeerRecord
	mu    sync.RWMutex
}

func main() {
	bs := &BootstrapServer{
		peers: make(map[string]*PeerRecord),
	}

	go bs.cleanupLoop()

	mux := http.NewServeMux()
	mux.HandleFunc("/register", bs.handleRegister)
	mux.HandleFunc("/peers", bs.handlePeers)
	mux.HandleFunc("/health", bs.handleHealth)

	log.Printf("Aether Bootstrap Server v%s listening on port %d", version, port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), mux))
}

// handleRegister receives a node announcing itself to the network
func (bs *BootstrapServer) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID   string `json:"id"`
		Port int    `json:"port"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" || req.Port == 0 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Use the client's IP + their declared port as the reachable address
	clientIP := extractIP(r.RemoteAddr)
	address := fmt.Sprintf("%s:%d", clientIP, req.Port)

	bs.mu.Lock()
	bs.peers[req.ID] = &PeerRecord{
		ID:       req.ID,
		Address:  address,
		LastSeen: time.Now(),
	}
	bs.mu.Unlock()

	log.Printf("Registered: %s at %s", req.ID, address)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "address": address})
}

// handlePeers returns the list of currently active nodes
func (bs *BootstrapServer) handlePeers(w http.ResponseWriter, r *http.Request) {
	// Allow nodes to exclude themselves by passing ?id=their-id
	excludeID := r.URL.Query().Get("id")

	bs.mu.RLock()
	peers := make([]*PeerRecord, 0, len(bs.peers))
	for _, p := range bs.peers {
		if p.ID != excludeID {
			peers = append(peers, p)
		}
	}
	bs.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"peers": peers,
		"count": len(peers),
	})
}

// handleHealth is a simple health check endpoint
func (bs *BootstrapServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	bs.mu.RLock()
	count := len(bs.peers)
	bs.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"version": version,
		"peers":   count,
	})
}

// cleanupLoop removes peers that haven't checked in recently
func (bs *BootstrapServer) cleanupLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		bs.mu.Lock()
		for id, p := range bs.peers {
			if time.Since(p.LastSeen) > peerTimeout {
				delete(bs.peers, id)
				log.Printf("Removed stale peer: %s", id)
			}
		}
		bs.mu.Unlock()
	}
}

func extractIP(remoteAddr string) string {
	// RemoteAddr is "ip:port" — strip the port
	if idx := strings.LastIndex(remoteAddr, ":"); idx != -1 {
		return remoteAddr[:idx]
	}
	return remoteAddr
}

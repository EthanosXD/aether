package main

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// Node is an Aether network participant
type Node struct {
	id        string
	startTime time.Time
	peers     map[string]*Peer
	mu        sync.RWMutex
	quit      chan struct{}
	server    *http.Server
	tlsCert   tls.Certificate
}

// Peer represents another node on the network
type Peer struct {
	ID      string    `json:"id"`
	Address string    `json:"address"`
	SeenAt  time.Time `json:"seen_at"`
}

func NewNode() *Node {
	return &Node{
		id:      generateID(),
		peers:   make(map[string]*Peer),
		quit:    make(chan struct{}),
		tlsCert: loadOrCreateTLS(),
	}
}

func (n *Node) Start() {
	n.startTime = time.Now()
	log.Printf("Node ID: %s", n.id)

	n.startPeerServer()
	n.startDiscovery()
	n.startBootstrap()
	n.startProxy()
	n.startExitServer()
	n.startDashboard()
}

func (n *Node) Stop() {
	close(n.quit)
	if n.server != nil {
		n.server.Close()
	}
}

func (n *Node) addPeer(p *Peer) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.peers[p.ID] = p
}

func (n *Node) removePeer(id string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	delete(n.peers, id)
}

func (n *Node) hasPeer(id string) bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	_, ok := n.peers[id]
	return ok
}

func (n *Node) getPeers() []*Peer {
	n.mu.RLock()
	defer n.mu.RUnlock()
	peers := make([]*Peer, 0, len(n.peers))
	for _, p := range n.peers {
		peers = append(peers, p)
	}
	return peers
}

func (n *Node) startDashboard() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", n.handleDashboard)
	mux.HandleFunc("/api/status", n.handleStatus)
	mux.HandleFunc("/api/peers", n.handlePeers)

	n.server = &http.Server{Addr: ":8080", Handler: mux}

	go func() {
		log.Println("Dashboard running at http://localhost:8080")
		if err := n.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Dashboard error: %v", err)
		}
	}()
}

func (n *Node) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	uptime := time.Since(n.startTime).Round(time.Second)
	fmt.Fprintf(w, `{"id":"%s","version":"%s","uptime":"%s","peers":%d,"status":"online"}`,
		n.id, version, uptime, len(n.getPeers()))
}

func (n *Node) handlePeers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	peers := n.getPeers()
	data, _ := json.Marshal(map[string]interface{}{"peers": peers, "count": len(peers)})
	w.Write(data)
}

func (n *Node) handleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, dashboardHTML)
}

func generateID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return "aether-" + hex.EncodeToString(b)
}

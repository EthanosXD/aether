package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
)

var bootstrapURL = flag.String("bootstrap", "https://aether-xcem.onrender.com", "Bootstrap server URL")

const registerInterval = 30 * time.Second

// startBootstrap registers with the bootstrap server and fetches initial peers
func (n *Node) startBootstrap() {
	if *bootstrapURL == "" {
		return
	}

	// Register in background so a slow/offline bootstrap server never blocks startup
	go func() {
		n.registerWithBootstrap()
		n.fetchPeersFromBootstrap()
	}()

	go func() {
		ticker := time.NewTicker(registerInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				n.registerWithBootstrap()
				n.fetchPeersFromBootstrap()
			case <-n.quit:
				return
			}
		}
	}()
}

func (n *Node) registerWithBootstrap() {
	body, _ := json.Marshal(map[string]interface{}{
		"id":   n.id,
		"port": peerPort,
	})

	resp, err := http.Post(
		fmt.Sprintf("%s/register", *bootstrapURL),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		log.Printf("Bootstrap register failed: %v", err)
		return
	}
	defer resp.Body.Close()
	log.Printf("Registered with bootstrap server: %s", *bootstrapURL)
}

func (n *Node) fetchPeersFromBootstrap() {
	url := fmt.Sprintf("%s/peers?id=%s", *bootstrapURL, n.id)
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var result struct {
		Peers []struct {
			ID      string `json:"id"`
			Address string `json:"address"`
		} `json:"peers"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return
	}

	for _, p := range result.Peers {
		if p.ID != n.id && !n.hasPeer(p.ID) {
			log.Printf("Bootstrap peer found: %s at %s", p.ID, p.Address)
			go n.connectToPeer(p.ID, p.Address)
		}
	}
}

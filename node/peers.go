package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

const (
	peerPort          = 42423
	heartbeatInterval = 30 * time.Second
	dialTimeout       = 5 * time.Second
)

type helloMsg struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Port    int    `json:"port"`
}

type peerListMsg struct {
	Peers []peerInfo `json:"peers"`
}

type peerInfo struct {
	ID      string `json:"id"`
	Address string `json:"address"`
}

// startPeerServer listens for incoming peer TCP connections
func (n *Node) startPeerServer() {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", peerPort))
	if err != nil {
		log.Fatalf("Peer server failed to start: %v", err)
	}

	log.Printf("Peer server listening on TCP port %d", peerPort)

	go func() {
		defer listener.Close()
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-n.quit:
					return
				default:
					continue
				}
			}
			go n.handleConnection(conn, "", true)
		}
	}()
}

// connectToPeer initiates an outbound connection to a known peer
func (n *Node) connectToPeer(id, addr string) {
	if n.hasPeer(id) {
		return
	}

	conn, err := net.DialTimeout("tcp", addr, dialTimeout)
	if err != nil {
		return
	}

	n.handleConnection(conn, addr, false)
}

// handleConnection runs the full peer handshake and keeps the connection alive
func (n *Node) handleConnection(conn net.Conn, addr string, incoming bool) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)

	// Step 1: Send our hello
	hello := helloMsg{ID: n.id, Version: version, Port: peerPort}
	data, _ := json.Marshal(hello)
	fmt.Fprintf(conn, "HELLO %s\n", data)

	// Step 2: Read their hello
	if !scanner.Scan() {
		return
	}
	line := scanner.Text()
	if !strings.HasPrefix(line, "HELLO ") {
		return
	}

	var theirHello helloMsg
	if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "HELLO ")), &theirHello); err != nil {
		return
	}

	if theirHello.ID == n.id {
		return // Connected to ourselves
	}

	if addr == "" {
		addr = fmt.Sprintf("%s:%d", conn.RemoteAddr().(*net.TCPAddr).IP, theirHello.Port)
	}

	peer := &Peer{ID: theirHello.ID, Address: addr, SeenAt: time.Now()}
	n.addPeer(peer)
	log.Printf("Peer connected: %s (%s)", peer.ID, peer.Address)

	// Step 3: Share our peer list
	n.sendPeerList(conn, theirHello.ID)

	// Step 4: Receive their peer list and connect to new ones
	if scanner.Scan() {
		n.receivePeerList(scanner.Text())
	}

	// Step 5: Heartbeat loop — keeps connection alive, detects disconnects
	n.runHeartbeat(conn, scanner, peer)
}

func (n *Node) sendPeerList(conn net.Conn, excludeID string) {
	peers := n.getPeers()
	infos := make([]peerInfo, 0, len(peers))
	for _, p := range peers {
		if p.ID != excludeID {
			infos = append(infos, peerInfo{ID: p.ID, Address: p.Address})
		}
	}
	data, _ := json.Marshal(peerListMsg{Peers: infos})
	fmt.Fprintf(conn, "PEERS %s\n", data)
}

func (n *Node) receivePeerList(line string) {
	if !strings.HasPrefix(line, "PEERS ") {
		return
	}
	var msg peerListMsg
	if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "PEERS ")), &msg); err != nil {
		return
	}
	for _, pi := range msg.Peers {
		if pi.ID != n.id && !n.hasPeer(pi.ID) {
			go n.connectToPeer(pi.ID, pi.Address)
		}
	}
}

func (n *Node) runHeartbeat(conn net.Conn, scanner *bufio.Scanner, peer *Peer) {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	msgs := make(chan string)
	go func() {
		for scanner.Scan() {
			msgs <- scanner.Text()
		}
		close(msgs)
	}()

	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				n.removePeer(peer.ID)
				log.Printf("Peer disconnected: %s", peer.ID)
				return
			}
			if msg == "PING" {
				fmt.Fprintf(conn, "PONG\n")
			}
			n.mu.Lock()
			peer.SeenAt = time.Now()
			n.mu.Unlock()
		case <-ticker.C:
			fmt.Fprintf(conn, "PING\n")
		case <-n.quit:
			return
		}
	}
}

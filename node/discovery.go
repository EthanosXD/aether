package main

import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

const (
	discoveryPort     = 42424
	discoveryInterval = 5 * time.Second
	discoveryTag      = "AETHER_NODE"
)

// startDiscovery broadcasts this node's presence on the LAN and listens for others
func (n *Node) startDiscovery() {
	go n.listenForPeers()
	go n.broadcastPresence()
	log.Printf("Peer discovery active on UDP port %d", discoveryPort)
}

// broadcastPresence announces this node to the local network every 5 seconds
func (n *Node) broadcastPresence() {
	broadcastAddr := &net.UDPAddr{
		IP:   net.IPv4bcast,
		Port: discoveryPort,
	}

	conn, err := net.ListenUDP("udp4", &net.UDPAddr{})
	if err != nil {
		log.Printf("Discovery broadcast error: %v", err)
		return
	}
	defer conn.Close()

	msg := []byte(fmt.Sprintf("%s %s %d", discoveryTag, n.id, peerPort))
	ticker := time.NewTicker(discoveryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			conn.WriteToUDP(msg, broadcastAddr)
		case <-n.quit:
			return
		}
	}
}

// listenForPeers listens for broadcast announcements from other nodes
func (n *Node) listenForPeers() {
	addr := &net.UDPAddr{Port: discoveryPort}
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		log.Printf("Discovery listen error: %v", err)
		return
	}
	defer conn.Close()

	buf := make([]byte, 256)
	for {
		select {
		case <-n.quit:
			return
		default:
		}

		conn.SetReadDeadline(time.Now().Add(time.Second))
		bytesRead, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}

		parts := strings.Fields(string(buf[:bytesRead]))
		if len(parts) != 3 || parts[0] != discoveryTag {
			continue
		}

		peerID := parts[1]
		peerTCPPort := parts[2]

		// Skip our own broadcasts
		if peerID == n.id {
			continue
		}

		// Skip already connected peers
		if n.hasPeer(peerID) {
			continue
		}

		peerAddr := fmt.Sprintf("%s:%s", remoteAddr.IP.String(), peerTCPPort)
		log.Printf("Discovered peer via LAN: %s at %s", peerID, peerAddr)
		go n.connectToPeer(peerID, peerAddr)
	}
}

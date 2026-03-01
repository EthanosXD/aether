package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"
)

const (
	proxyPort = 1080  // SOCKS5 — local apps connect here
	exitPort  = 42426 // Exit node — receives forwarded traffic from peers
)

// startProxy starts the local SOCKS5 proxy server (localhost only)
func (n *Node) startProxy() {
	listener, err := net.Listen("tcp", "127.0.0.1:1080")
	if err != nil {
		log.Printf("SOCKS5 proxy error: %v", err)
		return
	}
	log.Printf("SOCKS5 proxy on localhost:%d", proxyPort)

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
			go n.handleSOCKS5(conn)
		}
	}()
}

// startExitServer listens for traffic forwarded by other Aether peers
func (n *Node) startExitServer() {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", exitPort))
	if err != nil {
		log.Printf("Exit server error: %v", err)
		return
	}
	log.Printf("Exit server on port %d", exitPort)

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
			go n.handleExitRequest(conn)
		}
	}()
}

// handleSOCKS5 implements the SOCKS5 protocol handshake and routes traffic
func (n *Node) handleSOCKS5(client net.Conn) {
	defer client.Close()
	buf := make([]byte, 256)

	// --- Greeting ---
	if _, err := io.ReadFull(client, buf[:2]); err != nil {
		return
	}
	if buf[0] != 0x05 {
		return // Not SOCKS5
	}
	nMethods := int(buf[1])
	if _, err := io.ReadFull(client, buf[:nMethods]); err != nil {
		return
	}
	client.Write([]byte{0x05, 0x00}) // Accept, no auth

	// --- Connection request ---
	if _, err := io.ReadFull(client, buf[:4]); err != nil {
		return
	}
	if buf[1] != 0x01 { // Only CONNECT supported
		client.Write([]byte{0x05, 0x07, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}

	var host string
	switch buf[3] {
	case 0x01: // IPv4
		if _, err := io.ReadFull(client, buf[:4]); err != nil {
			return
		}
		host = net.IP(buf[:4]).String()
	case 0x03: // Domain name
		if _, err := io.ReadFull(client, buf[:1]); err != nil {
			return
		}
		length := int(buf[0])
		if _, err := io.ReadFull(client, buf[:length]); err != nil {
			return
		}
		host = string(buf[:length])
	case 0x04: // IPv6
		if _, err := io.ReadFull(client, buf[:16]); err != nil {
			return
		}
		host = net.IP(buf[:16]).String()
	default:
		return
	}

	if _, err := io.ReadFull(client, buf[:2]); err != nil {
		return
	}
	port := int(buf[0])<<8 | int(buf[1])
	dest := fmt.Sprintf("%s:%d", host, port)

	// --- Route traffic ---
	remote, via := n.dialDestination(dest)
	if remote == nil {
		client.Write([]byte{0x05, 0x05, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}
	defer remote.Close()

	log.Printf("Proxy: %s → %s (via %s)", client.RemoteAddr(), dest, via)
	client.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
	relay(client, remote)
}

// dialDestination tries to connect through a peer, falls back to direct
func (n *Node) dialDestination(dest string) (net.Conn, string) {
	peers := n.getPeers()
	for _, peer := range peers {
		conn, err := n.dialThroughPeer(peer, dest)
		if err == nil {
			return conn, peer.ID
		}
	}

	// No peers available — connect directly if this node has internet access
	conn, err := net.DialTimeout("tcp", dest, 10*time.Second)
	if err != nil {
		return nil, ""
	}
	return conn, "direct"
}

// dialThroughPeer opens a new connection to a peer's exit server and requests forwarding
func (n *Node) dialThroughPeer(peer *Peer, dest string) (net.Conn, error) {
	host, _, err := net.SplitHostPort(peer.Address)
	if err != nil {
		return nil, err
	}

	exitAddr := fmt.Sprintf("%s:%d", host, exitPort)
	conn, err := net.DialTimeout("tcp", exitAddr, 5*time.Second)
	if err != nil {
		return nil, err
	}

	fmt.Fprintf(conn, "AETHER_PROXY %s\n", dest)

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		conn.Close()
		return nil, fmt.Errorf("no response from peer exit")
	}
	conn.SetReadDeadline(time.Time{})

	response := scanner.Text()
	if !strings.HasPrefix(response, "OK") {
		conn.Close()
		return nil, fmt.Errorf("peer exit refused: %s", response)
	}

	return conn, nil
}

// handleExitRequest receives a forwarded connection from a peer and makes the real connection
func (n *Node) handleExitRequest(conn net.Conn) {
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		return
	}
	conn.SetReadDeadline(time.Time{})

	line := scanner.Text()
	if !strings.HasPrefix(line, "AETHER_PROXY ") {
		return
	}
	dest := strings.TrimPrefix(line, "AETHER_PROXY ")

	remote, err := net.DialTimeout("tcp", dest, 10*time.Second)
	if err != nil {
		fmt.Fprintf(conn, "ERR %v\n", err)
		log.Printf("Exit failed for %s: %v", dest, err)
		return
	}
	defer remote.Close()

	fmt.Fprintf(conn, "OK\n")
	log.Printf("Exit: relaying traffic to %s", dest)
	relay(conn, remote)
}

// relay copies traffic bidirectionally between two connections
func relay(a, b net.Conn) {
	done := make(chan struct{}, 2)
	go func() { io.Copy(a, b); done <- struct{}{} }()
	go func() { io.Copy(b, a); done <- struct{}{} }()
	<-done
}

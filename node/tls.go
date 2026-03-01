package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"os"
	"time"
)

const (
	certFile = "aether-cert.pem"
	keyFile  = "aether-key.pem"
)

// loadOrCreateTLS loads an existing TLS cert or generates a new one
func loadOrCreateTLS() tls.Certificate {
	// Try to load existing cert
	if _, err := os.Stat(certFile); err == nil {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err == nil {
			log.Println("Loaded existing TLS certificate")
			return cert
		}
	}

	// Generate a new ECDSA key pair
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatalf("TLS key generation failed: %v", err)
	}

	// Self-signed cert valid for 10 years
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{Organization: []string{"Aether Node"}},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		log.Fatalf("TLS cert generation failed: %v", err)
	}

	// Save to disk
	certOut, _ := os.Create(certFile)
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	certOut.Close()

	keyOut, _ := os.Create(keyFile)
	keyDER, _ := x509.MarshalECPrivateKey(key)
	pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	keyOut.Close()

	log.Println("Generated new TLS certificate")

	cert, _ := tls.LoadX509KeyPair(certFile, keyFile)
	return cert
}

// tlsServerConfig returns a TLS config for accepting peer connections
func tlsServerConfig(cert tls.Certificate) *tls.Config {
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequestClientCert, // Request but don't require (TOFU)
		MinVersion:   tls.VersionTLS13,
	}
}

// tlsClientConfig returns a TLS config for dialing peers
// We accept any cert (TOFU model — trust on first use, like SSH)
func tlsClientConfig(cert tls.Certificate) *tls.Config {
	return &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true, // TOFU: we verify by fingerprint, not CA chain
		MinVersion:         tls.VersionTLS13,
	}
}

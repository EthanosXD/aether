package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

const version = "0.1.0"

func main() {
	flag.Parse()
	log.Printf("Aether Node v%s starting...", version)

	node := NewNode()
	node.Start()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Aether Node shutting down...")
	node.Stop()
}

export PATH := $(PATH):/usr/local/go/bin

.PHONY: all node bootstrap run stop clean

all: node bootstrap

node:
	@echo "Building node..."
	@cd node && go build -o aether-node .
	@echo "✓ node/aether-node"

bootstrap:
	@echo "Building bootstrap server..."
	@cd bootstrap && go build -o aether-bootstrap .
	@echo "✓ bootstrap/aether-bootstrap"

run: all
	@echo "Starting bootstrap server..."
	@cd bootstrap && ./aether-bootstrap &
	@sleep 1
	@echo "Starting node..."
	@cd node && ./aether-node -bootstrap http://localhost:7070 &
	@echo ""
	@echo "✓ Aether running"
	@echo "  Dashboard  → http://localhost:8080"
	@echo "  SOCKS5     → localhost:1080"
	@echo "  Bootstrap  → http://localhost:7070/health"

stop:
	@pkill -f aether-node 2>/dev/null || true
	@pkill -f aether-bootstrap 2>/dev/null || true
	@echo "✓ Stopped"

clean: stop
	@rm -f node/aether-node bootstrap/aether-bootstrap
	@echo "✓ Cleaned"

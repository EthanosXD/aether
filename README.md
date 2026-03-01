# Aether

**P2P internet network — no ISP required, no censorship, free and open.**

Aether connects devices directly to each other. Traffic routes through a distributed network of nodes instead of a single ISP. The more nodes, the stronger and faster the network.

## Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/EthanosXD/aether/main/install.sh | sudo bash
```

Installs the Aether node and starts it as a system service. Then set your browser's SOCKS5 proxy to `localhost:1080`.

## How It Works

```
Your device                    Aether Network                   Internet
    │                              │                               │
    │──SOCKS5──► Node (yours)──────┼──► Peer node ────────────────► example.com
                     │             │
                     └─── Finds peers via:
                           1. UDP broadcast (LAN)
                           2. Bootstrap server (internet-wide)
                           3. Peer lists from connected nodes
```

- **No central authority** — bootstrap server is just a directory, not a proxy
- **No crypto / tokens** — plain subscriptions, no blockchain
- **Censorship resistant** — no single point to block or filter
- **Self-healing** — if nodes drop, traffic reroutes automatically

## Running Locally

**Requirements:** Go 1.23+

```bash
# Build everything
make all

# Start bootstrap server + node
make run

# Dashboard → http://localhost:8080
# SOCKS5 proxy → localhost:1080
# Bootstrap API → http://localhost:7070/health
```

## Project Structure

```
aether/
├── node/             Core P2P node (Go)
│   ├── main.go       Entry point
│   ├── node.go       Node struct and HTTP dashboard
│   ├── discovery.go  LAN peer discovery (UDP broadcast)
│   ├── peers.go      TCP peer connections and protocol
│   ├── bootstrap.go  Internet-wide peer discovery
│   ├── proxy.go      SOCKS5 proxy + traffic routing
│   └── dashboard.go  Web UI
├── bootstrap/        Peer directory server (Go)
├── api/              Subscription API — accounts, payments, license keys
├── docs/             Landing page (GitHub Pages)
└── install.sh        One-command installer
```

## Node Flags

```bash
./aether-node \
  -bootstrap http://bootstrap.aether.network:7070   # Bootstrap server URL
  -license   AETH-xxxx-xxxx-xxxx-xxxx               # Pro license key (optional)
```

## API Server

The API server handles user accounts and Pro subscriptions.

```bash
cd api
cp .env.example .env   # Fill in Stripe keys
go build -o aether-api .
./aether-api
```

Endpoints:
- `POST /api/signup` — create account
- `POST /api/login` — log in
- `GET  /api/me` — current user + subscription
- `GET  /api/license/verify?key=AETH-...` — verify a license key (called by nodes)
- `POST /api/checkout` — start Stripe checkout (Pro)
- `POST /api/webhook` — Stripe webhook

## Running Your Own Bootstrap Server

```bash
cd bootstrap
go build -o aether-bootstrap .
./aether-bootstrap   # Listens on port 7070
```

Then point nodes at it:
```bash
./aether-node -bootstrap http://your-server:7070
```

## Contributing

Run a node. Tell people. Open issues. Submit PRs.

Every node that joins makes the network faster and more resilient for everyone.

## License

MIT

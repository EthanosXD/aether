package main

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Aether Node</title>
  <style>
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body { background: #0a0a0f; color: #e0e0e0; font-family: 'Segoe UI', sans-serif; min-height: 100vh; }
    .header { padding: 24px 32px; border-bottom: 1px solid #1a1a2e; display: flex; align-items: center; gap: 12px; }
    .logo { font-size: 24px; font-weight: 700; color: #7c6af7; letter-spacing: -0.5px; }
    .version { font-size: 12px; color: #555; background: #111; padding: 2px 8px; border-radius: 4px; }
    .main { padding: 32px; max-width: 960px; }
    .card { background: #111; border: 1px solid #1a1a2e; border-radius: 12px; padding: 24px; margin-bottom: 24px; }
    .card-header { display: flex; align-items: center; gap: 10px; margin-bottom: 20px; }
    .dot { width: 10px; height: 10px; border-radius: 50%; background: #4ade80; animation: pulse 2s infinite; }
    @keyframes pulse { 0%,100%{opacity:1} 50%{opacity:0.4} }
    .card-title { font-size: 16px; font-weight: 600; }
    .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 16px; }
    .stat { background: #0a0a0f; border: 1px solid #1a1a2e; border-radius: 8px; padding: 16px; }
    .stat-label { font-size: 11px; color: #555; text-transform: uppercase; letter-spacing: 1px; margin-bottom: 8px; }
    .stat-value { font-size: 22px; font-weight: 700; color: #7c6af7; }
    .stat-sub { font-size: 11px; color: #444; margin-top: 4px; }
    .section-title { font-size: 12px; font-weight: 600; color: #555; text-transform: uppercase; letter-spacing: 1px; margin-bottom: 16px; }
    .peer-row { display: flex; justify-content: space-between; align-items: center; padding: 12px 16px; background: #0a0a0f; border: 1px solid #1a1a2e; border-radius: 8px; margin-bottom: 8px; }
    .peer-id { font-size: 14px; color: #ccc; font-family: monospace; }
    .peer-addr { font-size: 12px; color: #444; }
    .peer-status { font-size: 11px; color: #4ade80; background: #0d2718; padding: 2px 8px; border-radius: 4px; }
    .empty { color: #333; font-size: 14px; text-align: center; padding: 40px; }
    .ports { font-size: 12px; color: #444; margin-top: 4px; }
  </style>
</head>
<body>
  <div class="header">
    <span class="logo">Aether</span>
    <span class="version" id="ver">v0.1.0</span>
  </div>
  <div class="main">
    <div class="card">
      <div class="card-header">
        <div class="dot"></div>
        <span class="card-title">Node Online</span>
      </div>
      <div class="grid">
        <div class="stat">
          <div class="stat-label">Node ID</div>
          <div class="stat-value" id="node-id" style="font-size:13px;word-break:break-all;color:#aaa">—</div>
        </div>
        <div class="stat">
          <div class="stat-label">Uptime</div>
          <div class="stat-value" id="uptime">—</div>
        </div>
        <div class="stat">
          <div class="stat-label">Connected Peers</div>
          <div class="stat-value" id="peer-count">0</div>
          <div class="stat-sub">Aether nodes</div>
        </div>
        <div class="stat">
          <div class="stat-label">SOCKS5 Proxy</div>
          <div class="stat-value" style="font-size:14px;color:#4ade80">localhost:1080</div>
          <div class="stat-sub">point your browser here</div>
        </div>
      </div>
    </div>

    <div class="card" style="border-color:#1a2e1a">
      <div class="section-title">How to use Aether</div>
      <div style="font-size:13px;color:#666;line-height:1.8">
        Set your browser's SOCKS5 proxy to <span style="color:#4ade80;font-family:monospace">localhost:1080</span>.<br>
        Your traffic will route through the Aether network automatically.
      </div>
    </div>

    <div class="card">
      <div class="section-title">Connected Peers</div>
      <div id="peer-list"><div class="empty">No peers connected yet.<br>Other Aether nodes on your network will appear here automatically.</div></div>
    </div>
  </div>

  <script>
    async function refresh() {
      try {
        const [status, peersRes] = await Promise.all([
          fetch('/api/status').then(r => r.json()),
          fetch('/api/peers').then(r => r.json())
        ]);
        document.getElementById('ver').textContent = 'v' + status.version;
        document.getElementById('node-id').textContent = status.id;
        document.getElementById('uptime').textContent = status.uptime;
        document.getElementById('peer-count').textContent = status.peers;

        const list = document.getElementById('peer-list');
        if (peersRes.peers && peersRes.peers.length > 0) {
          list.innerHTML = peersRes.peers.map(p => {
            const seen = new Date(p.seen_at).toLocaleTimeString();
            return '<div class="peer-row">' +
              '<div><div class="peer-id">' + p.id + '</div><div class="peer-addr">' + p.address + '</div></div>' +
              '<span class="peer-status">online</span>' +
            '</div>';
          }).join('');
        } else {
          list.innerHTML = '<div class="empty">No peers connected yet.<br>Other Aether nodes on your network will appear here automatically.</div>';
        }
      } catch(e) {}
    }
    refresh();
    setInterval(refresh, 3000);
  </script>
</body>
</html>`

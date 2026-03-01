package main

import "fmt"

const baseStyle = `
<style>
* { box-sizing: border-box; margin: 0; padding: 0; }
body { background: #08080f; color: #e2e2e2; font-family: 'Segoe UI', system-ui, sans-serif; min-height: 100vh; display: flex; flex-direction: column; }
a { color: #a78bfa; text-decoration: none; }
a:hover { text-decoration: underline; }
nav { display: flex; justify-content: space-between; align-items: center; padding: 20px 40px; border-bottom: 1px solid #1a1a2e; }
.logo { font-size: 22px; font-weight: 700; color: #7c6af7; }
.card { background: #0f0f1a; border: 1px solid #1a1a2e; border-radius: 16px; padding: 40px; width: 100%; max-width: 440px; margin: auto; }
.card h1 { font-size: 24px; font-weight: 700; margin-bottom: 8px; }
.card .sub { color: #666; font-size: 14px; margin-bottom: 32px; }
label { display: block; font-size: 13px; color: #888; margin-bottom: 6px; }
input { width: 100%; background: #08080f; border: 1px solid #1a1a2e; border-radius: 8px; padding: 12px 14px; color: #e2e2e2; font-size: 15px; margin-bottom: 20px; outline: none; transition: border-color 0.2s; }
input:focus { border-color: #7c6af7; }
.btn { width: 100%; background: #7c6af7; color: #fff; border: none; padding: 13px; border-radius: 8px; font-size: 16px; font-weight: 600; cursor: pointer; transition: opacity 0.2s; }
.btn:hover { opacity: 0.88; }
.btn-outline { background: transparent; border: 1px solid #1a1a2e; color: #e2e2e2; }
.error { background: rgba(239,68,68,0.1); border: 1px solid rgba(239,68,68,0.3); color: #fca5a5; padding: 12px 16px; border-radius: 8px; font-size: 14px; margin-bottom: 20px; display: none; }
.footer-link { text-align: center; font-size: 14px; color: #555; margin-top: 24px; }
</style>`

const loginHTML = `<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8"><title>Login — Aether</title>` + baseStyle + `</head><body>
<nav><a class="logo" href="https://ethanosxd.github.io/aether/">Aether</a></nav>
<div class="card">
  <h1>Welcome back</h1>
  <p class="sub">Log in to your Aether account</p>
  <div class="error" id="err"></div>
  <form id="form">
    <label>Email</label>
    <input type="email" name="email" placeholder="you@example.com" required>
    <label>Password</label>
    <input type="password" name="password" placeholder="••••••••" required>
    <button class="btn" type="submit">Log in</button>
  </form>
  <p class="footer-link">No account? <a href="/signup">Sign up free</a></p>
</div>
<script>
document.getElementById('form').addEventListener('submit', async e => {
  e.preventDefault();
  const data = new FormData(e.target);
  const res = await fetch('/api/login', { method: 'POST', body: data });
  const json = await res.json();
  if (res.ok) { window.location = '/dashboard'; }
  else { const el = document.getElementById('err'); el.textContent = json.error; el.style.display = 'block'; }
});
</script></body></html>`

const signupHTML = `<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8"><title>Sign Up — Aether</title>` + baseStyle + `</head><body>
<nav><a class="logo" href="https://ethanosxd.github.io/aether/">Aether</a></nav>
<div class="card">
  <h1>Create your account</h1>
  <p class="sub">Free forever. Upgrade to Pro for premium features.</p>
  <div class="error" id="err"></div>
  <form id="form">
    <label>Email</label>
    <input type="email" name="email" placeholder="you@example.com" required>
    <label>Password</label>
    <input type="password" name="password" placeholder="Min. 8 characters" required minlength="8">
    <button class="btn" type="submit">Create account</button>
  </form>
  <p class="footer-link">Already have an account? <a href="/login">Log in</a></p>
</div>
<script>
document.getElementById('form').addEventListener('submit', async e => {
  e.preventDefault();
  const data = new FormData(e.target);
  const res = await fetch('/api/signup', { method: 'POST', body: data });
  const json = await res.json();
  if (res.ok) { window.location = '/dashboard'; }
  else { const el = document.getElementById('err'); el.textContent = json.error; el.style.display = 'block'; }
});
</script></body></html>`

func renderDashboard(sub *Subscription) string {
	isPro := sub.Tier == "pro" && sub.Status == "active"

	tierBadge := `<span style="background:#1a1a2e;color:#666;padding:3px 10px;border-radius:12px;font-size:12px">Free</span>`
	if isPro {
		tierBadge = `<span style="background:rgba(124,106,247,0.15);color:#a78bfa;padding:3px 10px;border-radius:12px;font-size:12px">Pro</span>`
	}

	licenseBlock := `<div style="color:#444;font-size:14px">Upgrade to Pro to get a license key.</div>`
	if isPro && sub.LicenseKey != "" {
		licenseBlock = fmt.Sprintf(`
		<div style="background:#08080f;border:1px solid #1a1a2e;border-radius:8px;padding:16px;font-family:monospace;font-size:15px;color:#4ade80;display:flex;justify-content:space-between;align-items:center">
			<span id="lk">%s</span>
			<button onclick="navigator.clipboard.writeText('%s');this.textContent='Copied!';setTimeout(()=>this.textContent='Copy',2000)"
				style="background:#1a1a2e;border:none;color:#888;padding:6px 12px;border-radius:6px;cursor:pointer;font-size:12px">Copy</button>
		</div>
		<p style="font-size:13px;color:#555;margin-top:10px">Run your node with: <code style="color:#7c6af7">./aether-node -license %s</code></p>`,
			sub.LicenseKey, sub.LicenseKey, sub.LicenseKey)
	}

	billingBtn := `<a href="/api/checkout" style="display:inline-block;background:#7c6af7;color:#fff;padding:12px 28px;border-radius:8px;font-weight:600;font-size:15px;text-decoration:none">Upgrade to Pro — $8/month</a>`
	if isPro {
		billingBtn = `<a href="/api/billing-portal" style="display:inline-block;background:transparent;border:1px solid #1a1a2e;color:#e2e2e2;padding:12px 28px;border-radius:8px;font-weight:500;font-size:15px;text-decoration:none">Manage Subscription</a>`
	}

	return `<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8"><title>Dashboard — Aether</title>` + baseStyle + `
<style>
.main { max-width: 700px; margin: 0 auto; padding: 40px 24px; }
.section { background: #0f0f1a; border: 1px solid #1a1a2e; border-radius: 16px; padding: 28px; margin-bottom: 20px; }
.section h2 { font-size: 16px; font-weight: 700; margin-bottom: 20px; display: flex; align-items: center; gap: 10px; }
.feature-row { display: flex; align-items: center; gap: 10px; font-size: 14px; color: #888; margin-bottom: 10px; }
.feature-row.on { color: #e2e2e2; }
.feature-row .icon { font-size: 16px; }
</style>
</head><body>
<nav>
  <a class="logo" href="/">Aether</a>
  <a href="/api/logout" style="font-size:14px;color:#555">Log out</a>
</nav>
<div class="main">

  <div class="section">
    <h2>Your Plan ` + tierBadge + `</h2>
    <div class="feature-row on"><span class="icon">✓</span> SOCKS5 proxy access</div>
    <div class="feature-row on"><span class="icon">✓</span> LAN peer discovery</div>
    <div class="feature-row on"><span class="icon">✓</span> Bootstrap server access</div>
    <div class="feature-row ` + proClass(isPro) + `"><span class="icon">` + tick(isPro) + `</span> Priority routing through fast nodes</div>
    <div class="feature-row ` + proClass(isPro) + `"><span class="icon">` + tick(isPro) + `</span> Unlimited bandwidth</div>
    <div class="feature-row ` + proClass(isPro) + `"><span class="icon">` + tick(isPro) + `</span> Dedicated exit nodes</div>
    <div style="margin-top:24px">` + billingBtn + `</div>
  </div>

  <div class="section">
    <h2>License Key</h2>
    ` + licenseBlock + `
  </div>

  <div class="section">
    <h2>Quick Start</h2>
    <p style="font-size:14px;color:#666;margin-bottom:16px">Get Aether running on any Linux machine:</p>
    <div style="background:#08080f;border:1px solid #1a1a2e;border-radius:8px;padding:16px;font-family:monospace;font-size:13px;color:#4ade80">
      curl -fsSL https://raw.githubusercontent.com/EthanosXD/aether/main/install.sh | sudo bash
    </div>
    <p style="font-size:13px;color:#555;margin-top:10px">Then set your browser's SOCKS5 proxy to <code style="color:#7c6af7">localhost:1080</code></p>
  </div>

</div>
</body></html>`
}

func proClass(isPro bool) string {
	if isPro {
		return "on"
	}
	return ""
}

func tick(isPro bool) string {
	if isPro {
		return "✓"
	}
	return "○"
}

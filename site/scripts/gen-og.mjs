// Generate 1200×630 OG/social cards in the ufi NOC brand style, one per page + a default.
// Renders a small HTML template with headless Chrome, then crops the top 1200×630 with sharp
// (render-tall, crop-top avoids Chrome viewport rounding/scrollbar artifacts). Run: `node scripts/gen-og.mjs`.
import { execSync } from "node:child_process";
import { mkdirSync, writeFileSync, rmSync, existsSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, resolve } from "node:path";
import sharp from "sharp";

const W = 1200, H = 630;

const __dir = dirname(fileURLToPath(import.meta.url));
const root = resolve(__dir, "..");
const outDir = resolve(root, "public/og");
const tmpDir = resolve(root, ".og-tmp");
mkdirSync(outDir, { recursive: true });
mkdirSync(tmpDir, { recursive: true });

const CHROME =
  process.env.CHROME_BIN ||
  ["google-chrome", "google-chrome-stable", "chromium", "chromium-browser"].find((b) => {
    try { execSync(`command -v ${b}`, { stdio: "ignore" }); return true; } catch { return false; }
  });
if (!CHROME) { console.error("No Chrome/Chromium found (set CHROME_BIN)."); process.exit(1); }

const pages = [
  { slug: "default", title: "An agent-friendly CLI for UniFi Network", sub: "Read-only by default · official local Integration API" },
  { slug: "index", title: "An agent-friendly CLI for Ubiquiti UniFi Network", sub: "Mutations gated · config reviewed by hash · bounded JSON" },
  { slug: "getting-started", title: "Getting started", sub: "Describe the binary offline, then point it at your console" },
  { slug: "auth", title: "Authentication & security", sub: "X-API-KEY · stdin → OS keyring · doctor · threat model" },
  { slug: "config", title: "Reviewed config — apply <hash>", sub: "--dry-run → plan + hash → ufi apply <hash>" },
  { slug: "commands", title: "Command reference", sub: "Every command, flag, and exit code — reads safe by default" },
  { slug: "agents", title: "For agents", sub: "Self-describing · fenced · mutation-gated · token-disciplined" },
];

const card = (title, sub) => `<!doctype html><html><head><meta charset="utf-8"/>
<style>
  @import url("https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@500;800&family=Inter+Tight:wght@600&display=swap");
  *{margin:0;box-sizing:border-box}
  html,body{width:${W}px;height:${H}px}
  body{font-family:"JetBrains Mono","Segoe UI",monospace;color:#f4f1e9;
    background:#0a0e14;position:relative;overflow:hidden;padding:72px 84px 92px;display:flex;flex-direction:column}
  .glow{position:absolute;inset:0;background:
    radial-gradient(50rem 30rem at 88% -8%, rgba(34,211,238,.18), transparent 60%),
    radial-gradient(40rem 30rem at -6% 18%, rgba(45,212,191,.14), transparent 55%)}
  .gridbg{position:absolute;inset:0;
    background-image:linear-gradient(rgba(34,211,238,.06) 1px,transparent 1px),linear-gradient(90deg,rgba(34,211,238,.06) 1px,transparent 1px);
    background-size:46px 46px;
    -webkit-mask-image:radial-gradient(120% 90% at 50% 0%,#000 35%,transparent 78%)}
  .brand{display:flex;align-items:center;gap:16px;position:relative}
  .brand svg{width:50px;height:50px}
  .brand b{font-size:40px;font-weight:800;letter-spacing:-.03em;font-family:"JetBrains Mono",monospace}
  .status{position:relative;margin-top:14px;font-size:18px;color:#34d399;letter-spacing:.05em}
  h1{position:relative;font-size:62px;font-weight:800;line-height:1.05;letter-spacing:-.035em;margin-top:auto;max-width:1020px;font-family:"JetBrains Mono",monospace}
  .sub{position:relative;color:#8497ad;font-size:24px;margin-top:22px}
  .pills{position:relative;display:flex;gap:14px;margin-top:28px;font-size:19px}
  .pill{border:1px solid #233143;border-radius:999px;padding:8px 18px;color:#c3cedd;background:#0c1119}
  .url{position:absolute;right:84px;bottom:40px;color:#22d3ee;font-size:23px}
  .grad{background:linear-gradient(100deg,#2dd4bf,#22d3ee 55%,#0ea5e9);-webkit-background-clip:text;background-clip:text;color:transparent}
</style></head><body>
  <div class="glow"></div>
  <div class="gridbg"></div>
  <div class="brand">
    <svg viewBox="0 0 32 32" fill="none"><linearGradient id="g" x1="2" y1="28" x2="30" y2="4" gradientUnits="userSpaceOnUse"><stop stop-color="#2dd4bf"/><stop offset=".55" stop-color="#22d3ee"/><stop offset="1" stop-color="#0ea5e9"/></linearGradient>
    <rect x="3" y="20" width="5" height="9" rx="1.5" fill="url(#g)" opacity=".55"/>
    <rect x="10.5" y="14" width="5" height="15" rx="1.5" fill="url(#g)" opacity=".7"/>
    <rect x="18" y="8" width="5" height="21" rx="1.5" fill="url(#g)" opacity=".85"/>
    <rect x="25.5" y="3" width="3.5" height="26" rx="1.5" fill="url(#g)"/></svg>
    <b>ufi</b>
  </div>
  <div class="status">● ONLINE — UniFi Network Integration API</div>
  <h1>${title.replace(/(UniFi Network|agents|hash&gt;|console)\.?$/, (m) => `<span class="grad">${m}</span>`)}</h1>
  <div class="sub">${sub}</div>
  <div class="pills"><span class="pill">read-only default</span><span class="pill">mutation-gated</span><span class="pill">injection-fenced</span></div>
  <div class="url">uficli.sh</div>
</body></html>`;

for (const p of pages) {
  const html = resolve(tmpDir, `${p.slug}.html`);
  const shot = resolve(tmpDir, `${p.slug}.raw.png`);
  const png = resolve(outDir, `${p.slug}.png`);
  // sub may contain a literal "<hash>" — escape angle brackets for HTML.
  const sub = p.sub.replace(/</g, "&lt;").replace(/>/g, "&gt;");
  const title = p.title.replace(/</g, "&lt;").replace(/>/g, "&gt;");
  writeFileSync(html, card(title, sub));
  // Render at 2× height, then crop the top W×H — avoids Chrome viewport rounding/scrollbar edges.
  execSync(
    `"${CHROME}" --headless=new --no-sandbox --disable-gpu --hide-scrollbars ` +
      `--force-device-scale-factor=1 --window-size=${W},${H * 2} --screenshot="${shot}" "file://${html}"`,
    { stdio: "ignore" }
  );
  if (!existsSync(shot)) { console.error("failed:", p.slug); process.exit(1); }
  await sharp(shot).extract({ left: 0, top: 0, width: W, height: H }).png().toFile(png);
  console.log("og:", p.slug + ".png");
}
rmSync(tmpDir, { recursive: true, force: true });
console.log("done →", outDir);

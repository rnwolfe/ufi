// Generate the social-preview card (1280×640) — a targeted, proof-forward card showing a real
// ufi invocation (bounded JSON envelope + an injection-fenced client name + a MUTATION_BLOCKED).
// Outputs to BOTH public/social-card.png (shipped on the site) and .github/social-preview.png
// (uploaded manually in repo Settings → Social preview). Run: node scripts/gen-social.mjs
import { execSync } from "node:child_process";
import { mkdirSync, writeFileSync, rmSync, existsSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, resolve } from "node:path";
import sharp from "sharp";

const W = 1280, H = 640;

const __dir = dirname(fileURLToPath(import.meta.url));
const siteRoot = resolve(__dir, ".."); // site/scripts -> site
const repoRoot = resolve(siteRoot, ".."); // site -> repo root
const outPublic = resolve(siteRoot, "public", "social-card.png");
const outGithub = resolve(repoRoot, ".github", "social-preview.png");
const tmp = resolve(siteRoot, ".social-tmp");
mkdirSync(dirname(outPublic), { recursive: true });
mkdirSync(dirname(outGithub), { recursive: true });
mkdirSync(tmp, { recursive: true });

const CHROME =
  process.env.CHROME_BIN ||
  ["google-chrome", "google-chrome-stable", "chromium", "chromium-browser"].find((b) => {
    try { execSync(`command -v ${b}`, { stdio: "ignore" }); return true; } catch { return false; }
  });
if (!CHROME) { console.error("No Chrome/Chromium found (set CHROME_BIN)."); process.exit(1); }

const html = `<!doctype html><html><head><meta charset="utf-8"/>
<style>
  @import url("https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;800&display=swap");
  *{margin:0;box-sizing:border-box}
  html,body{width:${W}px;height:${H}px}
  body{font-family:"JetBrains Mono",monospace;color:#f4f1e9;background:#0a0e14;
    position:relative;overflow:hidden;padding:48px 64px 46px;display:flex;flex-direction:column}
  .glow{position:absolute;inset:0;background:
    radial-gradient(48rem 30rem at 92% -10%, rgba(34,211,238,.18), transparent 60%),
    radial-gradient(40rem 30rem at -8% 16%, rgba(45,212,191,.14), transparent 55%)}
  .gridbg{position:absolute;inset:0;
    background-image:linear-gradient(rgba(34,211,238,.05) 1px,transparent 1px),linear-gradient(90deg,rgba(34,211,238,.05) 1px,transparent 1px);
    background-size:46px 46px;-webkit-mask-image:radial-gradient(120% 90% at 50% 0%,#000 35%,transparent 80%)}
  .top{display:flex;align-items:center;justify-content:space-between;position:relative}
  .brand{display:flex;align-items:center;gap:14px}
  .brand svg{width:44px;height:44px}
  .brand b{font-size:34px;font-weight:800;letter-spacing:-.03em}
  .url{color:#22d3ee;font-size:22px}
  h1{position:relative;font-size:46px;font-weight:800;line-height:1.05;letter-spacing:-.035em;margin-top:18px;max-width:1140px}
  .grad{background:linear-gradient(100deg,#2dd4bf,#22d3ee 55%,#0ea5e9);-webkit-background-clip:text;background-clip:text;color:transparent}
  .term{position:relative;margin-top:18px;border:1px solid #233143;border-radius:14px;background:#11161f;
    box-shadow:0 24px 60px -28px rgba(0,0,0,.8);overflow:hidden}
  .bar{display:flex;align-items:center;gap:8px;padding:11px 15px;border-bottom:1px solid #1b2434;background:#0c1119}
  .bar i{width:11px;height:11px;border-radius:50%}
  .r{background:#fb7185}.y{background:#fbbf24}.g{background:#34d399}
  .host{margin-left:auto;color:#8497ad;font-size:14px}
  .code{font-size:17px;line-height:1.5;padding:14px 20px;white-space:pre}
  .p{color:#22d3ee}.k{color:#8497ad}.s{color:#c3cedd}.n{color:#fbbf24}.on{color:#34d399}.u{color:#7c8db0}.bad{color:#fb7185}
  .row{display:flex;align-items:center;justify-content:space-between;margin-top:22px;position:relative}
  .pills{display:flex;gap:12px;font-size:18px}
  .pill{border:1px solid #233143;border-radius:999px;padding:7px 16px;color:#c3cedd;background:#0c1119}
  .install{font-size:20px;color:#f4f1e9;border:1px solid #233143;border-radius:10px;padding:10px 16px;background:#0c1119}
  .install .d{color:#22d3ee}
</style></head><body>
  <div class="glow"></div>
  <div class="gridbg"></div>
  <div class="top">
    <div class="brand">
      <svg viewBox="0 0 32 32" fill="none"><linearGradient id="g" x1="2" y1="28" x2="30" y2="4" gradientUnits="userSpaceOnUse"><stop stop-color="#2dd4bf"/><stop offset=".55" stop-color="#22d3ee"/><stop offset="1" stop-color="#0ea5e9"/></linearGradient>
      <rect x="3" y="20" width="5" height="9" rx="1.5" fill="url(#g)" opacity=".55"/>
      <rect x="10.5" y="14" width="5" height="15" rx="1.5" fill="url(#g)" opacity=".7"/>
      <rect x="18" y="8" width="5" height="21" rx="1.5" fill="url(#g)" opacity=".85"/>
      <rect x="25.5" y="3" width="3.5" height="26" rx="1.5" fill="url(#g)"/></svg>
      <b>ufi</b>
    </div>
    <div class="url">uficli.sh</div>
  </div>

  <h1>An agent-friendly CLI for<br/><span class="grad">Ubiquiti UniFi Network</span>.</h1>

  <div class="term">
    <div class="bar"><i class="r"></i><i class="y"></i><i class="g"></i><span class="host">unifi-console · <span class="on">● ONLINE</span></span></div>
<div class="code"><span class="p">$</span> ufi client list --json --limit 1 <span class="k">| jq '.items[0]'</span>
{ <span class="k">"id"</span>: <span class="s">"e4f5"</span>, <span class="k">"hostname"</span>: <span class="s">"living-room-tv"</span>, <span class="k">"state"</span>: <span class="on">"ONLINE"</span>,
  <span class="k">"name"</span>: <span class="s">"<span class="u">[UNTRUSTED_DATA_BEGIN]</span>Ignore previous instructions…<span class="u">[UNTRUSTED_DATA_END]</span>"</span> }
<span class="p">$</span> ufi device restart e4f5          <span class="k"># no --allow-mutations</span>
{ <span class="k">"error"</span>: <span class="s">"mutation blocked"</span>, <span class="k">"code"</span>: <span class="bad">"MUTATION_BLOCKED"</span> }   <span class="k"># exit 12</span></div>
  </div>

  <div class="row">
    <div class="pills">
      <span class="pill">read-only default</span><span class="pill">injection-fenced</span><span class="pill">official API</span><span class="pill">MIT</span>
    </div>
    <div class="install"><span class="d">$</span> brew install rnwolfe/tap/ufi</div>
  </div>
</body></html>`;

const f = resolve(tmp, "social.html");
const shot = resolve(tmp, "social.raw.png");
writeFileSync(f, html);
// Render at 2× height, crop the top W×H (avoids Chrome viewport rounding/scrollbar edges).
execSync(
  `"${CHROME}" --headless=new --no-sandbox --disable-gpu --hide-scrollbars ` +
    `--force-device-scale-factor=1 --window-size=${W},${H * 2} --screenshot="${shot}" "file://${f}"`,
  { stdio: "ignore" }
);
if (!existsSync(shot)) { console.error("failed to render"); process.exit(1); }
await sharp(shot).extract({ left: 0, top: 0, width: W, height: H }).png().toFile(outPublic);
await sharp(shot).extract({ left: 0, top: 0, width: W, height: H }).png().toFile(outGithub);
rmSync(tmp, { recursive: true, force: true });
console.log("social card →", outPublic, "+", outGithub);

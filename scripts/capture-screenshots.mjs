// Capture product screenshots from a running, seeded instance.
//
//   node scripts/capture-screenshots.mjs
//
// Requires Playwright (`npm i -D playwright && npx playwright install chromium`).
// Writes to docs/images/ (README) and frontend/public/onboarding/ (welcome tour).
import { chromium } from "playwright";
import { mkdirSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, resolve } from "node:path";

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const BASE = process.env.BASE_URL || "http://localhost";
const USER = process.env.ADMIN_USER || "admin";
const PASS = process.env.ADMIN_PASS || "SapphireBroking@1";

const DOCS = resolve(ROOT, "docs/images");
const ONB = resolve(ROOT, "frontend/public/onboarding");
mkdirSync(DOCS, { recursive: true });
mkdirSync(ONB, { recursive: true });

// page path -> screenshot basename
const SHOTS = [
  ["/dashboard", "dashboard"],
  ["/files", "files"],
  ["/teams", "teams"],
  ["/search?q=policy", "search"],
  ["/ask", "ask"],
  ["/common", "common"],
  ["/admin/audit", "audit"],
  ["/api-keys", "api"],
  ["/shared", "share"],
];

const run = async () => {
  const browser = await chromium.launch();
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 }, deviceScaleFactor: 2 });
  const page = await ctx.newPage();

  // Log in.
  await page.goto(`${BASE}/login`, { waitUntil: "networkidle" });
  await page.fill('input[name="identifier"], input[type="text"]', USER);
  await page.fill('input[type="password"]', PASS);
  // If another session is active, the app prompts to take over — auto-accept.
  page.on("dialog", (d) => d.accept().catch(() => {}));
  await page.click('button[type="submit"]');
  await page.waitForURL("**/dashboard", { timeout: 15000 }).catch(() => {});
  // Dismiss the first-login welcome tour if present.
  await page.getByText("Skip", { exact: true }).click({ timeout: 4000 }).catch(() => {});
  await page.waitForTimeout(800);

  for (const [path, name] of SHOTS) {
    try {
      await page.goto(`${BASE}${path}`, { waitUntil: "networkidle" });
      await page.waitForTimeout(1200); // let animations settle
      await page.screenshot({ path: resolve(DOCS, `${name}.png`) });
      // Onboarding images reuse the same captures.
      await page.screenshot({ path: resolve(ONB, `${name}.png`) });
      console.log("captured", name);
    } catch (e) {
      console.warn("skip", name, e.message);
    }
  }

  // Multilingual showcase: switch to Hindi and capture the Files screen.
  try {
    await page.evaluate(() => localStorage.setItem("sphr_locale", "hi"));
    await page.goto(`${BASE}/files`, { waitUntil: "networkidle" });
    await page.waitForTimeout(1200);
    await page.screenshot({ path: resolve(DOCS, "multilingual.png") });
    console.log("captured multilingual (hi)");
  } catch (e) {
    console.warn("skip multilingual", e.message);
  }

  await browser.close();
};

run().then(() => console.log("done")).catch((e) => { console.error(e); process.exit(1); });

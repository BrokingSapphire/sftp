// Copies the canonical /brand.config.json (repo root) into the frontend so it
// can be imported at build time. Runs automatically on predev/prebuild.
//
// The canonical file lives one level above the frontend project. In the Docker
// image it is copied into the project root by the Dockerfile, so we look in
// both places and use whichever exists.
import { existsSync, readFileSync, writeFileSync, mkdirSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const frontendRoot = join(here, "..");

const candidates = [
  join(frontendRoot, "..", "brand.config.json"), // repo root (local dev)
  join(frontendRoot, "brand.config.json"), // copied into project (Docker)
];

const src = candidates.find(existsSync);
if (!src) {
  console.warn("[sync-brand] no brand.config.json found; keeping existing config/brand.json");
  process.exit(0);
}

let parsed;
try {
  parsed = JSON.parse(readFileSync(src, "utf8"));
} catch (e) {
  console.error(`[sync-brand] ${src} is not valid JSON:`, e.message);
  process.exit(1);
}
// Strip $comment keys anywhere in the tree.
const clean = JSON.parse(JSON.stringify(parsed, (k, v) => (k === "$comment" ? undefined : v)));

// SECURITY: never ship SMTP or SSO credentials to the browser bundle. The
// frontend only needs to know whether Microsoft SSO is enabled (to show the
// button). Everything else stays server-side (backend reads the root file).
delete clean.smtp;
if (clean.sso?.microsoft) {
  clean.sso = { microsoft: { enabled: !!clean.sso.microsoft.enabled } };
}
if (clean.ai) {
  clean.ai = { enabled: !!clean.ai.enabled }; // ollama URL/models stay server-side
}
if (clean.editor) {
  clean.editor = { enabled: !!clean.editor.enabled }; // doc-server URL comes from the API, secret stays server-side
}

const out = join(frontendRoot, "config", "brand.json");
mkdirSync(dirname(out), { recursive: true });
writeFileSync(out, JSON.stringify(clean, null, 2) + "\n");
console.log(`[sync-brand] ${src} → config/brand.json`);

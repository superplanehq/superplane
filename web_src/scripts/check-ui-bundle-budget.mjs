import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import process from "node:process";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const distAssets = path.resolve(__dirname, "../../pkg/web/assets/dist/assets");

const MAX_ENTRY_BYTES = 3_500_000;
const REQUIRED_CHUNK_PATTERNS = [
  /^AppDefaultTabGate-.*\.js$/,
  /^AdminLayout-.*\.js$/,
  /^FactorySettingsRoutes-.*\.js$/,
  /^factories-.*\.js$/,
];

function fail(message) {
  console.error(`UI bundle budget: ${message}`);
  process.exit(1);
}

if (!fs.existsSync(distAssets)) {
  fail(`missing ${distAssets}. Run npm run build first.`);
}

const jsFiles = fs
  .readdirSync(distAssets)
  .filter((name) => name.endsWith(".js") && !name.endsWith(".map.js"));

if (jsFiles.length < 20) {
  fail(`expected route-split JS chunks, found ${jsFiles.length}`);
}

for (const pattern of REQUIRED_CHUNK_PATTERNS) {
  if (!jsFiles.some((name) => pattern.test(name))) {
    fail(`missing required lazy chunk ${pattern}`);
  }
}

const entry = jsFiles.find((name) => /^app-.*\.js$/.test(name));
if (!entry) {
  fail("missing app entry chunk");
}

const entryBytes = fs.statSync(path.join(distAssets, entry)).size;
if (entryBytes > MAX_ENTRY_BYTES) {
  fail(`${entry} is ${entryBytes} bytes; budget is ${MAX_ENTRY_BYTES}`);
}

console.log(
  `UI bundle budget: ${jsFiles.length} JS chunks, ${entry} ${entryBytes} bytes (limit ${MAX_ENTRY_BYTES})`,
);

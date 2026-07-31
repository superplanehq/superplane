import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const webRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const sourceRoot = path.join(webRoot, "src");
const baselinePath = path.join(webRoot, ".semantic-color-budget-baseline.json");
const shouldUpdateBaseline = process.argv.includes("--update-baseline");

const ignoredDirectoryNames = new Set(["api-client", "__fixtures__", "__tests__", "stories", "tests"]);
const ignoredFilePatterns = [/\.spec\.[jt]sx?$/, /\.stories\.[jt]sx?$/, /\.test\.[jt]sx?$/];
const paletteUtilityPattern =
  /(?:bg|text|border|outline|ring|divide|placeholder|fill|stroke)-(?:slate|gray|zinc|neutral|stone)(?:-\d{2,3})?(?:\/[\d.]+)?\b|(?:bg|text|border|outline|ring|divide|placeholder|fill|stroke)-(?:white|black)(?:\/[\d.]+)?\b/g;

const currentCounts = collectPaletteUtilityCounts(sourceRoot);

if (shouldUpdateBaseline) {
  writeBaseline(currentCounts);
  process.stdout.write(`Updated semantic color debt baseline (${totalCount(currentCounts)} utilities).\n`);
  process.exit(0);
}

const baseline = readBaseline();
const regressions = [];

for (const [file, currentCount] of Object.entries(currentCounts)) {
  const allowedCount = baseline.counts[file] ?? 0;
  if (currentCount > allowedCount) {
    regressions.push({ file, currentCount, allowedCount });
  }
}

if (regressions.length > 0) {
  process.stderr.write("Hardcoded neutral palette debt increased. Use semantic color utilities instead:\n");
  for (const regression of regressions) {
    process.stderr.write(
      `  ${regression.file}: ${regression.currentCount} utilities (baseline ${regression.allowedCount})\n`,
    );
  }
  process.stderr.write(
    "\nChoose a semantic role such as bg-surface-raised, text-content-secondary, or border-edge-default.\n",
  );
  process.exit(1);
}

process.stdout.write(
  `Semantic color debt is within budget (${totalCount(currentCounts)} remaining; baseline ${totalCount(baseline.counts)}).\n`,
);

function collectPaletteUtilityCounts(directory) {
  const counts = {};

  for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
    const absolutePath = path.join(directory, entry.name);

    if (entry.isDirectory()) {
      if (!ignoredDirectoryNames.has(entry.name)) {
        Object.assign(counts, collectPaletteUtilityCounts(absolutePath));
      }
      continue;
    }

    if (!entry.isFile() || !/\.[jt]sx?$/.test(entry.name) || ignoredFilePatterns.some((pattern) => pattern.test(entry.name))) {
      continue;
    }

    const source = fs.readFileSync(absolutePath, "utf8");
    const count = source.match(paletteUtilityPattern)?.length ?? 0;
    if (count === 0) {
      continue;
    }

    counts[path.relative(webRoot, absolutePath)] = count;
  }

  return counts;
}

function readBaseline() {
  if (!fs.existsSync(baselinePath)) {
    throw new Error(`Semantic color baseline is missing: ${baselinePath}`);
  }

  return JSON.parse(fs.readFileSync(baselinePath, "utf8"));
}

function writeBaseline(counts) {
  const sortedCounts = Object.fromEntries(Object.entries(counts).sort(([left], [right]) => left.localeCompare(right)));
  fs.writeFileSync(baselinePath, `${JSON.stringify({ version: 1, counts: sortedCounts }, null, 2)}\n`);
}

function totalCount(counts) {
  return Object.values(counts).reduce((total, count) => total + count, 0);
}

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import process from "node:process";
import { ESLint } from "eslint";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const webRoot = path.resolve(__dirname, "..");
const baselinePath = path.join(webRoot, ".eslint-budget-baseline.json");
const isUpdateBaseline = process.argv.includes("--update-baseline");
const ignoredPrefixes = ["src/api-client/", "storybook-static/", "dist/", "dist-ssr/", "node_modules/"];
const redStart = process.env.NO_COLOR ? "" : "\x1b[31m";
const colorEnd = process.env.NO_COLOR ? "" : "\x1b[0m";
const disallowedDisableNextLinePattern = /(?:\/\/|\/\*)\s*eslint-disable-next-line\b/;

function normalizeRuleId(message) {
  if (message.ruleId) {
    return message.ruleId;
  }

  if (message.fatal) {
    return "fatal";
  }

  return "unknown";
}

function toRelativeFilePath(filePath) {
  return path.relative(webRoot, filePath);
}

function extractIssues(results) {
  const issues = [];

  for (const result of results) {
    const filePath = toRelativeFilePath(result.filePath);
    const isIgnoredPath = ignoredPrefixes.some((prefix) => filePath.startsWith(prefix));
    const isStorybookStoryFile = filePath.endsWith(".stories.tsx");
    const isStorybookSupportFile = filePath.includes("/storybooks/");
    if (isIgnoredPath || isStorybookStoryFile || isStorybookSupportFile) {
      continue;
    }

    for (const message of result.messages) {
      issues.push({
        filePath,
        line: message.line ?? 0,
        column: message.column ?? 0,
        severity: message.severity === 2 ? "error" : "warning",
        ruleId: normalizeRuleId(message),
        message: message.message,
      });
    }
  }

  return issues;
}

function extractDisallowedDirectiveIssues(results) {
  const issues = [];

  for (const result of results) {
    const filePath = toRelativeFilePath(result.filePath);
    const isIgnoredPath = ignoredPrefixes.some((prefix) => filePath.startsWith(prefix));
    const isStorybookStoryFile = filePath.endsWith(".stories.tsx");
    const isStorybookSupportFile = filePath.includes("/storybooks/");
    if (isIgnoredPath || isStorybookStoryFile || isStorybookSupportFile) {
      continue;
    }

    let source;
    try {
      source = fs.readFileSync(result.filePath, "utf8");
    } catch {
      continue;
    }

    const lines = source.split(/\r?\n/u);
    for (const [index, line] of lines.entries()) {
      const matchIndex = line.search(disallowedDisableNextLinePattern);
      if (matchIndex === -1) {
        continue;
      }

      issues.push({
        filePath,
        line: index + 1,
        column: matchIndex + 1,
        severity: "error",
        ruleId: "no-eslint-disable-next-line",
        message: "Using eslint-disable-next-line is not allowed.",
      });
    }
  }

  return issues;
}

function summarizeByRule(issues) {
  const counts = {};
  for (const issue of issues) {
    counts[issue.ruleId] = (counts[issue.ruleId] ?? 0) + 1;
  }

  return Object.fromEntries(Object.entries(counts).sort((a, b) => b[1] - a[1]));
}

function extractRuleMetricValue(issue) {
  if (issue.ruleId === "max-lines") {
    const match = issue.message.match(/too many lines \((\d+)\)/iu);
    return match ? Number(match[1]) : null;
  }

  if (issue.ruleId === "complexity") {
    const match = issue.message.match(/complexity of (\d+)/iu);
    return match ? Number(match[1]) : null;
  }

  return null;
}

function summarizeMetricTotalsByRule(issues) {
  const totals = {};

  for (const issue of issues) {
    const metricValue = extractRuleMetricValue(issue);
    if (metricValue === null) {
      continue;
    }

    totals[issue.ruleId] = (totals[issue.ruleId] ?? 0) + metricValue;
  }

  return Object.fromEntries(Object.entries(totals).sort((a, b) => b[1] - a[1]));
}

function printRuleCountsVsBudget(currentByRule, maxAllowedByRule) {
  const allRuleIds = new Set([
    ...Object.keys(currentByRule),
    ...Object.keys(maxAllowedByRule),
  ]);

  const sortedRuleIds = [...allRuleIds].sort((a, b) => {
    const currentA = currentByRule[a] ?? 0;
    const currentB = currentByRule[b] ?? 0;
    if (currentA !== currentB) {
      return currentB - currentA;
    }

    return a.localeCompare(b);
  });

  if (sortedRuleIds.length === 0) {
    console.log("- No per-rule data found.");
    return;
  }

  for (const ruleId of sortedRuleIds) {
    const current = currentByRule[ruleId] ?? 0;
    const allowed = maxAllowedByRule[ruleId] ?? 0;
    const overBudget = current > allowed;
    const status = overBudget ? " !!! OVER BUDGET" : "";
    const line = `- ${ruleId}: ${current}/${allowed}${status}`;
    console.log(overBudget ? `${redStart}${line}${colorEnd}` : line);
  }
}

function printMetricTotalsVsBudget(currentByRule, maxAllowedByRule) {
  const allRuleIds = new Set([
    ...Object.keys(currentByRule),
    ...Object.keys(maxAllowedByRule),
  ]);

  const sortedRuleIds = [...allRuleIds].sort((a, b) => {
    const currentA = currentByRule[a] ?? 0;
    const currentB = currentByRule[b] ?? 0;
    if (currentA !== currentB) {
      return currentB - currentA;
    }

    return a.localeCompare(b);
  });

  if (sortedRuleIds.length === 0) {
    console.log("- No metric totals found.");
    return;
  }

  for (const ruleId of sortedRuleIds) {
    const current = currentByRule[ruleId] ?? 0;
    const allowed = maxAllowedByRule[ruleId] ?? 0;
    const overBudget = current > allowed;
    const status = overBudget ? " !!! OVER BUDGET" : "";
    const line = `- ${ruleId}: ${current}/${allowed}${status}`;
    console.log(overBudget ? `${redStart}${line}${colorEnd}` : line);
  }
}

function printIssues(issues) {
  if (issues.length === 0) {
    console.log("- No ESLint issues found.");
    return;
  }

  for (const issue of issues) {
    console.log(`- ${issue.filePath}:${issue.line}:${issue.column} [${issue.severity}] (${issue.ruleId}) ${issue.message}`);
  }
}

function readBaseline() {
  const raw = fs.readFileSync(baselinePath, "utf8");
  return JSON.parse(raw);
}

function writeBaseline(issues, countsByRule) {
  const metricTotalsByRule = summarizeMetricTotalsByRule(issues);

  const baseline = {
    maxAllowedTotalIssues: issues.length,
    maxAllowedByRule: countsByRule,
    maxAllowedMetricTotalByRule: metricTotalsByRule,
    updatedAt: new Date().toISOString(),
  };

  fs.writeFileSync(baselinePath, `${JSON.stringify(baseline, null, 2)}\n`, "utf8");
}

function findRegressions(currentByRule, baselineByRule) {
  const regressions = [];
  const currentEntries = Object.entries(currentByRule);

  for (const [ruleId, currentCount] of currentEntries) {
    const allowedCount = baselineByRule[ruleId] ?? 0;
    if (currentCount > allowedCount) {
      regressions.push({ ruleId, currentValue: currentCount, allowedValue: allowedCount });
    }
  }

  return regressions.sort((a, b) => b.currentValue - a.currentValue);
}

async function main() {
  const eslint = new ESLint({ cwd: webRoot });
  const results = await eslint.lintFiles(["."]);
  const issues = [...extractIssues(results), ...extractDisallowedDirectiveIssues(results)];
  const countsByRule = summarizeByRule(issues);
  const metricTotalsByRule = summarizeMetricTotalsByRule(issues);

  if (isUpdateBaseline) {
    writeBaseline(issues, countsByRule);
    console.log(`Updated ESLint budget baseline to ${issues.length} issue(s).`);
    console.log("All issues:");
    printIssues(issues);
    console.log("");
    console.log("");
    console.log("");
    console.log("Rule counts vs budget:");
    printRuleCountsVsBudget(countsByRule, countsByRule);
    console.log("");
    console.log("");
    console.log("");
    console.log("Rule metric totals vs budget:");
    printMetricTotalsVsBudget(metricTotalsByRule, metricTotalsByRule);
    console.log("");
    console.log("");
    console.log("");
    console.log(`WITHIN BUDGET ${issues.length}/${issues.length}`);
    return;
  }

  const baseline = readBaseline();
  const maxAllowedTotal = baseline.maxAllowedTotalIssues;
  const maxAllowedByRule = baseline.maxAllowedByRule ?? {};
  const maxAllowedMetricTotalByRule = baseline.maxAllowedMetricTotalByRule ?? {};

  const totalRegression = issues.length - maxAllowedTotal;
  const byRuleRegressions = findRegressions(countsByRule, maxAllowedByRule);
  const metricRegressions = findRegressions(metricTotalsByRule, maxAllowedMetricTotalByRule);

  if (totalRegression > 0 || byRuleRegressions.length > 0 || metricRegressions.length > 0) {
    console.error("ESLint budget exceeded.");
    console.error(`- Total issues: ${issues.length} (allowed ${maxAllowedTotal})`);

    if (byRuleRegressions.length > 0) {
      console.error("- Rule regressions:");
      for (const regression of byRuleRegressions.slice(0, 20)) {
        console.error(`  - ${regression.ruleId}: ${regression.currentValue} (allowed ${regression.allowedValue})`);
      }

      if (byRuleRegressions.length > 20) {
        console.error(`  ... and ${byRuleRegressions.length - 20} more`);
      }
    }

    if (metricRegressions.length > 0) {
      console.error("- Rule metric total regressions:");
      for (const regression of metricRegressions.slice(0, 20)) {
        console.error(`  - ${regression.ruleId}: ${regression.currentValue} (allowed ${regression.allowedValue})`);
      }

      if (metricRegressions.length > 20) {
        console.error(`  ... and ${metricRegressions.length - 20} more`);
      }
    }

    console.error("All issues:");
    printIssues(issues);
    console.error("");
    console.error("");
    console.error("");
    console.error("Rule counts vs budget:");
    printRuleCountsVsBudget(countsByRule, maxAllowedByRule);
    console.error("");
    console.error("");
    console.error("");
    console.error("Rule metric totals vs budget:");
    printMetricTotalsVsBudget(metricTotalsByRule, maxAllowedMetricTotalByRule);
    console.error("");
    console.error("");
    console.error("");
    console.error(`FAILED ${issues.length}/${maxAllowedTotal}`);
    process.exit(1);
  }

  console.log("All issues:");
  printIssues(issues);
  console.log("");
  console.log("");
  console.log("");
  console.log("Rule counts vs budget:");
  printRuleCountsVsBudget(countsByRule, maxAllowedByRule);
  console.log("");
  console.log("");
  console.log("");
  console.log("Rule metric totals vs budget:");
  printMetricTotalsVsBudget(metricTotalsByRule, maxAllowedMetricTotalByRule);
  console.log("");
  console.log("");
  console.log("");
  console.log(`WITHIN BUDGET ${issues.length}/${maxAllowedTotal}`);
}

main().catch((error) => {
  console.error("Failed to run ESLint budget check.");
  console.error(error);
  process.exit(1);
});

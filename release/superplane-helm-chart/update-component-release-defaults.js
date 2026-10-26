#!/usr/bin/env node

/* eslint-disable no-console */

const { execFileSync } = require("node:child_process");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");

const { parseVersion } = require("../component-release");

const API_VERSION = "2022-11-28";
const USER_AGENT = "superplane-component-release";
const RELEASE_REQUEST_TIMEOUT_SECONDS = 20;
const RELEASE_REQUEST_TIMEOUT_GRACE_MS = 5000;
const FALLBACK_COMMENT =
  "# Saved versions are the fallback when release lookup fails.";
const COMPONENTS = [
  {
    label: "Runner",
    tagPrefix: "runner/",
    missingMessage: "no stable runner release",
    apply: setRunnerDefaults,
  },
  {
    label: "Fleet Manager",
    tagPrefix: "fleet-manager/",
    missingMessage: "no stable Fleet Manager release",
    apply: setFleetManagerDefault,
  },
];

function nextPage(linkHeader) {
  if (!linkHeader) {
    return undefined;
  }
  for (const link of linkHeader.split(",")) {
    const match = link.match(/<([^>]+)>;\s*rel="([^"]+)"/);
    if (match && match[2] === "next") {
      return match[1];
    }
  }
  return undefined;
}

function linkHeader(headers) {
  const links = [];
  for (const line of headers.split(/\r?\n/)) {
    const match = /^link:\s*(.*)$/i.exec(line);
    if (match) {
      links.push(match[1]);
    }
  }
  return links.join(", ");
}

function requestReleases(pageURL, token) {
  const directory = fs.mkdtempSync(
    path.join(os.tmpdir(), "chart-component-releases-"),
  );
  const bodyPath = path.join(directory, "body");
  const headerPath = path.join(directory, "headers");
  try {
    const requestHeaderPath = path.join(directory, "request-headers");
    const requestHeaders = [
      "Accept: application/vnd.github+json",
      `User-Agent: ${USER_AGENT}`,
      `X-GitHub-Api-Version: ${API_VERSION}`,
    ];
    if (token) {
      requestHeaders.push(`Authorization: Bearer ${token}`);
    }
    fs.writeFileSync(requestHeaderPath, `${requestHeaders.join("\n")}\n`, {
      mode: 0o600,
    });

    const args = [
      "--silent",
      "--show-error",
      "--max-time",
      String(RELEASE_REQUEST_TIMEOUT_SECONDS),
      "--output",
      bodyPath,
      "--dump-header",
      headerPath,
      "--write-out",
      "%{http_code}",
      "--header",
      `@${requestHeaderPath}`,
      "--",
      pageURL,
    ];

    let statusCode;
    try {
      statusCode = execFileSync("curl", args, {
        encoding: "utf8",
        timeout:
          RELEASE_REQUEST_TIMEOUT_SECONDS * 1000 +
          RELEASE_REQUEST_TIMEOUT_GRACE_MS,
        killSignal: "SIGKILL",
      }).trim();
    } catch (error) {
      if (error && (error.code === "ETIMEDOUT" || error.status === 28)) {
        throw new Error("failed to list GitHub releases: request timed out");
      }
      throw new Error("failed to list GitHub releases");
    }

    const status = Number(statusCode);
    if (status !== 200) {
      throw new Error(
        `list GitHub releases failed with HTTP ${statusCode || "unknown"}`,
      );
    }

    const body = fs.readFileSync(bodyPath, "utf8");
    let releases;
    try {
      releases = JSON.parse(body);
    } catch {
      throw new Error("list GitHub releases returned invalid JSON");
    }
    if (!Array.isArray(releases)) {
      throw new Error("list GitHub releases returned invalid JSON");
    }

    return {
      releases,
      nextPageURL: nextPage(linkHeader(fs.readFileSync(headerPath, "utf8"))),
    };
  } finally {
    fs.rmSync(directory, { recursive: true, force: true });
  }
}

function listReleases(token) {
  const apiURL = (
    process.env.GITHUB_API_URL || "https://api.github.com"
  ).replace(/\/+$/, "");
  const repository = process.env.GITHUB_REPOSITORY || "superplanehq/superplane";
  let pageURL = `${apiURL}/repos/${repository}/releases?per_page=100`;
  const visitedPages = new Set();
  const releases = [];

  while (pageURL) {
    if (visitedPages.has(pageURL)) {
      throw new Error("GitHub releases pagination contains a cycle");
    }
    visitedPages.add(pageURL);
    const page = requestReleases(pageURL, token);
    releases.push(...page.releases);
    pageURL = page.nextPageURL;
  }
  return releases;
}

function stableVersion(release, tagPrefix) {
  if (!release || typeof release.tag_name !== "string") {
    return undefined;
  }
  if (release.draft === true || release.prerelease === true) {
    return undefined;
  }
  if (!release.tag_name.startsWith(tagPrefix)) {
    return undefined;
  }

  const version = release.tag_name.slice(tagPrefix.length);
  try {
    parseVersion(version);
  } catch {
    return undefined;
  }
  if (version.includes("-")) {
    return undefined;
  }
  return version;
}

function compareVersions(left, right) {
  const leftParts = left.slice(1).split(".").map(Number);
  const rightParts = right.slice(1).split(".").map(Number);
  for (let index = 0; index < 3; index += 1) {
    if (leftParts[index] !== rightParts[index]) {
      return leftParts[index] - rightParts[index];
    }
  }
  return 0;
}

function latestStableVersion(releases, tagPrefix) {
  let selected;
  for (const release of releases) {
    const version = stableVersion(release, tagPrefix);
    if (!version) {
      continue;
    }
    if (!selected || compareVersions(version, selected) > 0) {
      selected = version;
    }
  }
  return selected;
}

function chartLines(text) {
  const newlineAtEnd = text.endsWith("\n");
  const lines = text.split("\n");
  if (lines[lines.length - 1] === "") {
    lines.pop();
  }
  return { lines, newlineAtEnd };
}

function joinChartLines(lines, newlineAtEnd) {
  return lines.join("\n") + (newlineAtEnd ? "\n" : "");
}

function sectionBounds(lines) {
  const start = lines.findIndex((line) => line === "componentReleases:");
  if (start < 0) {
    throw new Error("componentReleases is missing from chart values");
  }
  let end = lines.length;
  for (let index = start + 1; index < lines.length; index += 1) {
    const line = lines[index];
    if (line === "" || line.startsWith(" ") || line.startsWith("\t")) {
      continue;
    }
    end = index;
    break;
  }
  return { start, end };
}

function updateSection(text, edit) {
  const { lines, newlineAtEnd } = chartLines(text);
  const { start, end } = sectionBounds(lines);
  const updated = edit(lines.slice(start, end));
  return joinChartLines(
    [...lines.slice(0, start), ...updated, ...lines.slice(end)],
    newlineAtEnd,
  );
}

function componentReleaseSection(text) {
  const { lines } = chartLines(text);
  const { start, end } = sectionBounds(lines);
  return lines.slice(start, end);
}

function reuseComponentReleases(sourceText, targetText) {
  const sourceSection = componentReleaseSection(sourceText);
  return updateSection(targetText, () => sourceSection);
}

function setRunnerDefaults(text, version) {
  return updateSection(text, (section) => {
    const next = section.slice();
    const versionIndex = next.findIndex((line) =>
      /^  runnerVersion:/.test(line),
    );
    if (versionIndex < 0) {
      throw new Error("runnerVersion is missing from chart values");
    }
    next[versionIndex] = `  runnerVersion: ${version}`;

    const listIndex = next.findIndex((line) =>
      /^  runnerVersions:\s*$/.test(line),
    );
    if (listIndex < 0) {
      throw new Error("runnerVersions is missing from chart values");
    }
    let listEnd = listIndex + 1;
    while (listEnd < next.length && /^    - /.test(next[listEnd])) {
      listEnd += 1;
    }
    if (listEnd === listIndex + 1) {
      throw new Error("runnerVersions list is missing from chart values");
    }
    next.splice(
      listIndex,
      listEnd - listIndex,
      "  runnerVersions:",
      `    - ${version}`,
    );
    return next;
  });
}

function setFleetManagerDefault(text, version) {
  return updateSection(text, (section) => {
    const index = section.findIndex((line) =>
      /^  fleetManagerVersion:/.test(line),
    );
    if (index < 0) {
      throw new Error("fleetManagerVersion is missing from chart values");
    }
    const next = section.slice();
    next[index] = `  fleetManagerVersion: ${version}`;
    return next;
  });
}

function ensureFallbackComment(text) {
  if (text.includes(FALLBACK_COMMENT)) {
    return text;
  }
  return text.replace(
    /^componentReleases:/m,
    `${FALLBACK_COMMENT}\ncomponentReleases:`,
  );
}

function writeValues(filePath, text) {
  const temporary = path.join(
    path.dirname(filePath),
    `.${path.basename(filePath)}.${process.pid}.tmp`,
  );
  fs.writeFileSync(temporary, text);
  fs.renameSync(temporary, filePath);
}

function applySelectedReleases(text, releases) {
  let next = text;
  let changed = false;
  for (const component of COMPONENTS) {
    try {
      const version = latestStableVersion(releases, component.tagPrefix);
      if (!version) {
        throw new Error(component.missingMessage);
      }
      next = component.apply(next, version);
      changed = true;
      console.log(`* ${component.label} default: ${version}`);
    } catch (error) {
      console.error(`Error: ${error.message}`);
    }
  }
  if (!changed) {
    return text;
  }
  return ensureFallbackComment(next);
}

function main() {
  if (process.argv[2] === "reuse") {
    const sourcePath = process.argv[3];
    const targetPath = process.argv[4];
    if (!sourcePath || !targetPath) {
      throw new Error(
        "reuse requires candidate and target chart values paths",
      );
    }
    if (!fs.existsSync(sourcePath)) {
      throw new Error(`candidate chart values not found: ${sourcePath}`);
    }
    if (!fs.existsSync(targetPath)) {
      throw new Error(`target chart values not found: ${targetPath}`);
    }

    const sourceText = fs.readFileSync(sourcePath, "utf8");
    const targetText = fs.readFileSync(targetPath, "utf8");
    writeValues(
      targetPath,
      reuseComponentReleases(sourceText, targetText),
    );
    console.log("* Reused candidate component release defaults");
    return;
  }

  const valuesPath =
    process.argv[2] || path.join(__dirname, "helm", "values.yaml");
  if (!fs.existsSync(valuesPath)) {
    throw new Error(`chart values not found: ${valuesPath}`);
  }

  let releases;
  try {
    releases = listReleases(process.env.GITHUB_TOKEN?.trim() || undefined);
  } catch (error) {
    console.error(`Error: ${error.message}`);
    return;
  }

  const text = fs.readFileSync(valuesPath, "utf8");
  const updated = applySelectedReleases(text, releases);
  if (updated !== text) {
    writeValues(valuesPath, updated);
  }
}

if (require.main === module) {
  try {
    main();
  } catch (error) {
    console.error(`Error: ${error.message}`);
    process.exit(1);
  }
}

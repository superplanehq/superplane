#!/usr/bin/env node
/**
 * Fix type errors from formatTimeAgo -> renderTimeAgo migration.
 * Updates function return types and variable types to support React.ReactNode.
 */
import fs from "fs";
import path from "path";

const ROOT = path.resolve("web_src/src");

function findFiles(dir, results = []) {
  const entries = fs.readdirSync(dir, { withFileTypes: true });
  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      findFiles(fullPath, results);
    } else if (
      (entry.name.endsWith(".ts") || entry.name.endsWith(".tsx")) &&
      !fullPath.includes("node_modules")
    ) {
      const content = fs.readFileSync(fullPath, "utf-8");
      if (content.includes("renderTimeAgo")) {
        results.push(fullPath);
      }
    }
  }
  return results;
}

const files = findFiles(ROOT);
console.log(`Found ${files.length} files to check`);

let updatedCount = 0;

for (const filePath of files) {
  let content = fs.readFileSync(filePath, "utf-8");
  const original = content;

  // Fix function return types: ): string { and ): string => that return renderTimeAgo
  // Pattern: functions like "buildSubtitle(...):" that return string but now return ReactNode
  content = content.replace(
    /function buildSubtitle\(([^)]*)\): string \{/g,
    "function buildSubtitle($1): string | React.ReactNode {"
  );

  // Fix getTitleAndSubtitle return types in trigger renderers
  content = content.replace(
    /getTitleAndSubtitle[^{]*\): \{ title: string; subtitle: string \}/g,
    (match) => match.replace("subtitle: string }", "subtitle: string | React.ReactNode }")
  );

  // Fix getTitleAndSubtitle return types (variant with createdAt: string)
  content = content.replace(
    /\): \{ title: string; subtitle: string; createdAt: string \}/g,
    "): { title: string; subtitle: string | React.ReactNode; createdAt: string }"
  );

  // Fix helper functions that have local function returning string
  // Pattern: "const generateEventSubtitle = (): string =>"  (already handled by first script)
  // Pattern: "function X(...): string {"
  // We need to be more targeted here - only change functions that contain renderTimeAgo

  // Fix local subtitle variables typed as "let subtitle: string = ..." or "const subtitle: string = ..."
  // These are trickier - let's look for specific patterns

  // Fix "subtitle: string" in return type annotations of local functions
  content = content.replace(
    /function getEventSubtitle\([^)]*\): string \{/g,
    (match) => {
      return match.replace("): string {", "): string | React.ReactNode {");
    }
  );

  // Fix common helper functions that return subtitle-like strings
  const helperPatterns = [
    /function buildSubtitle\([^)]*\): string \{/g,
    /function getSubtitle\([^)]*\): string \{/g,
    /function generateSubtitle\([^)]*\): string \{/g,
    /function getSummary\([^)]*\): string \{/g,
    /function buildSummary\([^)]*\): string \{/g,
  ];

  for (const pattern of helperPatterns) {
    content = content.replace(pattern, (match) => {
      return match.replace("): string {", "): string | React.ReactNode {");
    });
  }

  // Add React import if needed for React.ReactNode
  if (content.includes("React.ReactNode") && !content.includes("import React")) {
    content = content.replace(
      /^(import .+?;\n)/m,
      `$1import React from "react";\n`
    );
  }

  if (content !== original) {
    fs.writeFileSync(filePath, content);
    updatedCount++;
    console.log(`Updated: ${path.relative(process.cwd(), filePath)}`);
  }
}

console.log(`\nUpdated ${updatedCount} files`);

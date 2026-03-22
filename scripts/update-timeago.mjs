#!/usr/bin/env node
/**
 * Script to replace formatTimeAgo with renderTimeAgo/renderWithTimeAgo
 * across all mapper and utility files.
 */
import fs from "fs";
import path from "path";

const ROOT = path.resolve("web_src/src");

// Find all files that import formatTimeAgo
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
      if (content.includes("formatTimeAgo") && !fullPath.includes("components/TimeAgo")) {
        results.push(fullPath);
      }
    }
  }
  return results;
}

// Files we already handled manually
const SKIP_FILES = new Set([
  path.resolve("web_src/src/ui/componentBase/index.tsx"),
  path.resolve("web_src/src/ui/chainItem/ChainItem.tsx"),
  path.resolve("web_src/src/ui/componentSidebar/pages/ExecutionChainPage.tsx"),
  path.resolve("web_src/src/utils/date.ts"),
]);

const files = findFiles(ROOT).filter((f) => !SKIP_FILES.has(f));
console.log(`Found ${files.length} files to update`);

let updatedCount = 0;

for (const filePath of files) {
  let content = fs.readFileSync(filePath, "utf-8");
  const original = content;

  // Skip files that only use formatTimeAgo with .replace(" ago", "")
  // Check if all usages are .replace patterns
  const allUsages = content.match(/formatTimeAgo\([^)]+\)/g) || [];
  const replaceUsages = content.match(/formatTimeAgo\([^)]+\)\.replace/g) || [];
  if (allUsages.length > 0 && allUsages.length === replaceUsages.length) {
    // All usages are .replace patterns - skip this file
    continue;
  }

  // Track if we need renderTimeAgo and/or renderWithTimeAgo
  let needsRenderTimeAgo = false;
  let needsRenderWithTimeAgo = false;
  let needsReact = false;
  let keepFormatTimeAgo = replaceUsages.length > 0;

  // Pattern: subtitle return type ): string {
  content = content.replace(
    /subtitle\(context: SubtitleContext\): string \{/g,
    "subtitle(context: SubtitleContext): string | React.ReactNode {"
  );

  // Pattern: generateEventSubtitle functions with (): string
  content = content.replace(
    /const generateEventSubtitle = \(\): string => \{/g,
    "const generateEventSubtitle = (): string | React.ReactNode => {"
  );
  content = content.replace(
    /const generateEventSubtitle = \(\): string =>/g,
    "const generateEventSubtitle = (): string | React.ReactNode =>"
  );

  // Replace direct formatTimeAgo calls that are NOT followed by .replace
  // Pattern: formatTimeAgo(new Date(X)) NOT followed by .replace
  content = content.replace(
    /formatTimeAgo\(([^)]+)\)(?!\.replace)/g,
    (match, args) => {
      needsRenderTimeAgo = true;
      return `renderTimeAgo(${args})`;
    }
  );

  // Check if we need React import for the return type annotations
  if (content.includes("React.ReactNode") && !content.includes("import React")) {
    needsReact = true;
  }

  // Update the import statement
  if (needsRenderTimeAgo || needsRenderWithTimeAgo) {
    const renderImports = [];
    if (needsRenderTimeAgo) renderImports.push("renderTimeAgo");
    if (needsRenderWithTimeAgo) renderImports.push("renderWithTimeAgo");

    if (keepFormatTimeAgo) {
      // Keep formatTimeAgo import and add renderTimeAgo import
      content = content.replace(
        /import \{ formatTimeAgo \} from "@\/utils\/date";/,
        `import { formatTimeAgo } from "@/utils/date";\nimport { ${renderImports.join(", ")} } from "@/components/TimeAgo";`
      );
    } else {
      // Replace formatTimeAgo import with renderTimeAgo import
      content = content.replace(
        /import \{ formatTimeAgo \} from "@\/utils\/date";/,
        `import { ${renderImports.join(", ")} } from "@/components/TimeAgo";`
      );
    }
  }

  // Add React import if needed
  if (needsReact) {
    // Check if there's already a React import
    if (!content.includes("import React")) {
      // Add after the first import
      content = content.replace(
        /^(import .+?;\n)/m,
        `$1import React from "react";\n`
      );
    }
  }

  if (content !== original) {
    fs.writeFileSync(filePath, content);
    updatedCount++;
    console.log(`Updated: ${path.relative(process.cwd(), filePath)}`);
  }
}

console.log(`\nUpdated ${updatedCount} files`);

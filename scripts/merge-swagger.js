#!/usr/bin/env node

/**
 * Merge manually-defined swagger with auto-generated swagger
 * Used by protoc_openapi_spec.sh to combine auth/account endpoints with protobuf-generated API
 */

const fs = require("fs");

function mergeSwagger(generatedPath, manualPath, outputPath) {
  // Read both swagger files
  const generated = JSON.parse(fs.readFileSync(generatedPath, "utf8"));
  const manual = JSON.parse(fs.readFileSync(manualPath, "utf8"));

  // Merge paths
  generated.paths = {
    ...generated.paths,
    ...manual.paths,
  };

  // Merge definitions
  generated.definitions = {
    ...generated.definitions,
    ...manual.definitions,
  };

  // Merge tags (avoid duplicates)
  const existingTags = new Set(generated.tags.map((t) => t.name));
  for (const tag of manual.tags) {
    if (!existingTags.has(tag.name)) {
      generated.tags.push(tag);
    }
  }

  // Merge security definitions
  if (manual.securityDefinitions) {
    generated.securityDefinitions = {
      ...(generated.securityDefinitions || {}),
      ...manual.securityDefinitions,
    };
  }

  // Write merged output
  fs.writeFileSync(outputPath, JSON.stringify(generated, null, 2));

  console.log("Successfully merged swagger files");
}

// Get file paths from command line arguments
const [generatedPath, manualPath, outputPath] = process.argv.slice(2);

if (!generatedPath || !manualPath || !outputPath) {
  console.error(
    "Usage: merge-swagger.js <generated.json> <manual.json> <output.json>",
  );
  process.exit(1);
}

// Check if manual swagger exists
if (!fs.existsSync(manualPath)) {
  console.log(`Manual swagger not found at ${manualPath}, skipping merge`);
  process.exit(0);
}

try {
  mergeSwagger(generatedPath, manualPath, outputPath);
} catch (error) {
  console.error("Error merging swagger files:", error.message);
  process.exit(1);
}

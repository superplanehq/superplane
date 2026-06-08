#!/usr/bin/env node
/**
 * Inject the generated `ConsolePanelContent` JSON Schema into
 * `api/swagger/superplane.swagger.json`.
 *
 * The proto for `Console.Panel.content` is `google.protobuf.Value`, which
 * means `protoc-gen-openapiv2` renders the schema as an empty object (`{}`).
 * That hides the documented panel-content shapes from anyone reading the
 * OpenAPI spec, the Go/TS SDKs, or any downstream code generator.
 *
 * Swagger 2.0 (the format `protoc-gen-openapiv2` emits) doesn't support
 * `anyOf` / `oneOf`, so we can't drop the discriminated union straight
 * into `definitions/` — `openapi-generator-cli` would reject the spec. We
 * attach the full schema as a single vendor extension (`x-content-schema`)
 * on the `ConsolePanel` definition instead. Vendor extensions are passed
 * through unchanged by all OpenAPI tooling we use today, so the spec keeps
 * validating cleanly while the schema is still discoverable for anyone
 * reading the document, generating docs, or building a custom validator.
 *
 * The script is intentionally pure-Node with zero dependencies so it can
 * run in the dev container (`make openapi.spec.gen`) and in CI without an
 * extra npm install. Run with `node scripts/inject-console-panel-content-schema.mjs`.
 */

import { readFileSync, writeFileSync, existsSync } from "node:fs";
import { resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = resolve(__dirname, "..");

const SWAGGER_PATH = resolve(REPO_ROOT, "api/swagger/superplane.swagger.json");
const SCHEMA_PATH = resolve(REPO_ROOT, "api/schemas/console-panel-content.schema.json");
// The protoc-gen-openapiv2 generator derives definition names from the
// fully-qualified proto message name. After bullet 1 the `Console.Panel`
// renders as `ConsolePanel`; we only target that one.
const TARGET_DEFINITION = "ConsolePanel";

function die(message) {
  console.error(`inject-console-panel-content-schema: ${message}`);
  process.exit(1);
}

if (!existsSync(SWAGGER_PATH)) die(`swagger file not found at ${SWAGGER_PATH}`);
if (!existsSync(SCHEMA_PATH)) {
  die(
    `schema file not found at ${SCHEMA_PATH}. Did you forget to run 'cd web_src && npm run generate:console-schema'?`,
  );
}

const swagger = JSON.parse(readFileSync(SWAGGER_PATH, "utf8"));
const schema = JSON.parse(readFileSync(SCHEMA_PATH, "utf8"));

const definitions = swagger.definitions;
if (!definitions || typeof definitions !== "object") {
  die("swagger has no `definitions` section, nothing to inject into.");
}

const panel = definitions[TARGET_DEFINITION];
if (!panel) {
  die(
    `expected definition '${TARGET_DEFINITION}' in swagger; got [${Object.keys(definitions).join(", ")}]. ` +
      "If the panel definition was renamed during proto regeneration, update TARGET_DEFINITION.",
  );
}

if (!panel.properties || !panel.properties.content) {
  die(`definition '${TARGET_DEFINITION}' has no 'content' property to replace.`);
}

// Approach:
// - Update the `content` property to carry a human-readable description so
//   the swagger-ui and Go/TS SDKs at least know where the schema lives.
// - Attach the full JSON Schema as `x-content-schema` (the marker used by
//   downstream tooling), and a sibling `x-content-schema-ref` pointing to
//   the file in this repo for tools that prefer fetching the schema by URL.
//
// We can't inline the schema as a regular OpenAPI sub-schema because:
//   1. Swagger 2.0 (the format protoc-gen-openapiv2 emits) doesn't support
//      `anyOf`/`oneOf`, so openapi-generator-cli rejects the spec.
//   2. SDK generators that *do* crawl vendor extensions (`@hey-api/openapi-ts`)
//      follow `$ref` pointers, and the embedded schema's internal refs
//      collide with the swagger's top-level `definitions/` namespace.
//
// Encoding the schema as a JSON string sidesteps both problems: it's
// completely opaque to OpenAPI tooling but parseable by anyone who wants
// to validate panel content against the documented shape.
panel.properties.content = {
  description:
    "Polymorphic panel content. Shape is driven by `Panel.type`. The full discriminated " +
    "union of supported panel kinds lives in `api/schemas/console-panel-content.schema.json` " +
    "(generated from `web_src/src/pages/app/console/schema/panelContent.ts`); the same " +
    "JSON Schema is mirrored as `x-content-schema` on this definition for tools that prefer " +
    "to read it inline.",
};

panel["x-content-schema-ref"] = "api/schemas/console-panel-content.schema.json";
panel["x-content-schema"] = JSON.stringify(schema);

writeFileSync(SWAGGER_PATH, JSON.stringify(swagger, null, 2) + "\n");
const variantCount = Array.isArray(schema?.definitions?.ConsolePanelContent?.anyOf)
  ? schema.definitions.ConsolePanelContent.anyOf.length
  : 0;
console.log(
  `inject-console-panel-content-schema: injected ConsolePanelContent schema onto ${TARGET_DEFINITION} ` +
    `(${variantCount} panel-kind variants, encoded as stringified x-content-schema).`,
);

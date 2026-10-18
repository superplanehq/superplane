import { mkdtemp, mkdir, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";
import { describe, expect, it } from "bun:test";
import { build } from "vite";

import { keepWebAppManifestLinkOnPageOrigin } from "./webAppManifestLink";

const assetHost = "https://assets.example/releases/sha";
const assetBaseUrl = `${assetHost}/`;

function pageHtml(manifestLink: string): string {
  return `<!doctype html>
<html>
  <head>
    ${manifestLink}
    <link rel="apple-touch-icon" href="${assetHost}/assets/pwa/apple-touch-icon.png" />
    <link rel="icon" type="image/x-icon" href="${assetHost}/favicon.ico" />
    <link rel="stylesheet" crossorigin href="${assetHost}/assets/index.css" />
  </head>
  <body>
    <script type="module" crossorigin src="${assetHost}/assets/index.js"></script>
  </body>
</html>`;
}

function manifestHref(html: string): string | undefined {
  return linkHref(html, "manifest");
}

function linkHref(html: string, rel: string): string | undefined {
  const tag = html.match(new RegExp(`<link\\b(?=[^>]*\\brel=(["'])${rel}\\1)[^>]*>`, "i"))?.[0];
  return tag?.match(/\bhref=(["'])([^"']*)\1/i)?.[2];
}

function scriptSrc(html: string): string | undefined {
  return html.match(/<script\b[^>]*\bsrc=(["'])([^"']*)\1/i)?.[2];
}

describe("keepWebAppManifestLinkOnPageOrigin", () => {
  it("moves an asset-host manifest link back to the page origin", () => {
    const result = keepWebAppManifestLinkOnPageOrigin(
      pageHtml(`<link rel="manifest" href="${assetHost}/manifest.webmanifest" />`),
    );

    expect(manifestHref(result)).toBe("/manifest.webmanifest");
    expect(result).toContain(`href="${assetHost}/assets/pwa/apple-touch-icon.png"`);
    expect(result).toContain(`src="${assetHost}/assets/index.js"`);
    expect(result).toContain(`href="${assetHost}/favicon.ico"`);
    expect(result).toContain(`href="${assetHost}/assets/index.css"`);
    expect(result).not.toContain(`${assetHost}/manifest.webmanifest`);
  });

  it("rewrites the manifest link when the href comes before rel", () => {
    const result = keepWebAppManifestLinkOnPageOrigin(
      pageHtml(`<link href="${assetHost}/manifest.webmanifest" rel="manifest">`),
    );

    expect(manifestHref(result)).toBe("/manifest.webmanifest");
    expect(result).toContain(`src="${assetHost}/assets/index.js"`);
  });
});

describe("vite build manifest link", () => {
  it("keeps the built manifest on the page origin when the asset base is absolute", async () => {
    const html = await buildFixtureIndexHtml();

    expect(manifestHref(html)).toBe("/manifest.webmanifest");
    expect(html).not.toContain(`${assetBaseUrl}manifest.webmanifest`);
    expect(linkHref(html, "apple-touch-icon")).toBe(`${assetBaseUrl}assets/pwa/apple-touch-icon.png`);
    expect(linkHref(html, "icon")).toBe(`${assetBaseUrl}favicon.ico`);
    expect(linkHref(html, "stylesheet")?.startsWith(assetBaseUrl)).toBe(true);
    expect(scriptSrc(html)?.startsWith(assetBaseUrl)).toBe(true);
    expect(html).toContain('content="web-app-manifest-build"');
  }, 120_000);
});

async function buildFixtureIndexHtml(): Promise<string> {
  const root = await mkdtemp(path.join(tmpdir(), "web-app-manifest-"));
  const previousAssetBaseUrl = process.env.VITE_ASSET_BASE_URL;
  process.env.VITE_ASSET_BASE_URL = assetHost;

  try {
    await writeBuildFixture(root);
    await build({
      configFile: path.resolve(import.meta.dirname, "../../vite.config.ts"),
      root,
      logLevel: "silent",
      build: {
        outDir: path.join(root, "dist"),
        emptyOutDir: true,
        sourcemap: false,
      },
    });

    return await readFile(path.join(root, "dist", "index.html"), "utf8");
  } finally {
    restoreAssetBaseUrl(previousAssetBaseUrl);
    await rm(root, { recursive: true, force: true });
  }
}

async function writeBuildFixture(root: string): Promise<void> {
  await mkdir(path.join(root, "public/assets/pwa"), { recursive: true });
  await mkdir(path.join(root, "src"), { recursive: true });
  await writeFile(path.join(root, "public/favicon.ico"), "ico");
  await writeFile(path.join(root, "public/manifest.webmanifest"), "{}\n");
  await writeFile(path.join(root, "public/assets/pwa/apple-touch-icon.png"), "png");
  await writeFile(path.join(root, "src/main.ts"), 'import "./styles.css";\n');
  await writeFile(path.join(root, "src/styles.css"), "body { color: black; }\n");
  await writeFile(
    path.join(root, "index.html"),
    `<!doctype html>
<html>
  <head>
    <meta name="fixture" content="web-app-manifest-build" />
    <link rel="icon" type="image/x-icon" href="/favicon.ico" />
    <link rel="manifest" href="/manifest.webmanifest" />
    <link rel="apple-touch-icon" href="/assets/pwa/apple-touch-icon.png" />
    <link rel="stylesheet" href="/src/styles.css" />
  </head>
  <body>
    <script type="module" src="/src/main.ts"></script>
  </body>
</html>`,
  );
}

function restoreAssetBaseUrl(previousAssetBaseUrl: string | undefined): void {
  if (previousAssetBaseUrl === undefined) {
    delete process.env.VITE_ASSET_BASE_URL;
    return;
  }

  process.env.VITE_ASSET_BASE_URL = previousAssetBaseUrl;
}

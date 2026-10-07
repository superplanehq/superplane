import { describe, expect, it } from "bun:test";

import { keepWebAppManifestLinkOnPageOrigin } from "./webAppManifestLink";

const assetHost = "https://assets.example/releases/sha";

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
  const tag = html.match(/<link\b[^>]*\brel=(["'])manifest\1[^>]*>/i)?.[0];
  return tag?.match(/\bhref=(["'])([^"']*)\1/i)?.[2];
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

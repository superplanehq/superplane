const pageOriginManifestHref = 'href="/manifest.webmanifest"';

export function keepWebAppManifestLinkOnPageOrigin(html: string): string {
  const manifestLinkTag = /<link\b[^>]*\brel=(["'])manifest\1[^>]*>/gi;
  const hrefAttribute = /\bhref\s*=\s*(["'])[^"']*\1/i;

  return html.replace(manifestLinkTag, (tag) => tag.replace(hrefAttribute, pageOriginManifestHref));
}

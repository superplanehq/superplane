import templateManifest from "../../../../templates/manifest.json";
import type { AppEntry } from "./AppDetailModal";

export const APP_CATALOG: AppEntry[] = templateManifest;

export function filterAppCatalog(search: string, includeTags = true): AppEntry[] {
  const query = search.trim().toLowerCase();
  if (!query) return APP_CATALOG;

  return APP_CATALOG.filter(
    (app) =>
      app.title.toLowerCase().includes(query) ||
      app.description.toLowerCase().includes(query) ||
      app.integrations.some((integration) => integration.toLowerCase().includes(query)) ||
      (includeTags && app.tags.some((tag) => tag.toLowerCase().includes(query))),
  );
}

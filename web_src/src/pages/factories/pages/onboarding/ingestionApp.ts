import type { FactoryApp } from "@/api-client";
import { getFactoryDefinition, INGESTION_FACTORY_ID, SENTRY_INGESTION_FACTORY_ID } from "@/pages/home/factories";

// Workspaces set up before the app was renamed still carry the old title.
const LEGACY_INGESTION_TITLE = "Issue Intake";

/**
 * Resolve the ingestion app among a workspace's automations. The apps API
 * exposes only id, name, and description, so the bundled title is the one
 * stable signal we have.
 */
export function findIngestionApp(apps: FactoryApp[]): FactoryApp | undefined {
  return findFactoryApp(apps, INGESTION_FACTORY_ID, [LEGACY_INGESTION_TITLE]);
}

export function installedIngestionFactoryIds(apps: FactoryApp[]): Set<string> {
  const ids = [INGESTION_FACTORY_ID, SENTRY_INGESTION_FACTORY_ID];
  return new Set(ids.filter((factoryId) => findIngestionFactoryApp(apps, factoryId)));
}

export function findIngestionFactoryApp(apps: FactoryApp[], factoryId: string): FactoryApp | undefined {
  const legacyTitles = factoryId === INGESTION_FACTORY_ID ? [LEGACY_INGESTION_TITLE] : [];
  return findFactoryApp(apps, factoryId, legacyTitles);
}

function findFactoryApp(apps: FactoryApp[], factoryId: string, legacyTitles: string[] = []): FactoryApp | undefined {
  const titles = [getFactoryDefinition(factoryId).title, ...legacyTitles].map(normalizeAppName);
  return apps.find((app) => titles.includes(normalizeAppName(app.name)));
}

// Onboarding appends " (2)", " (3)", … when a canvas name is already taken.
function normalizeAppName(name: string | null | undefined): string {
  return (name ?? "")
    .trim()
    .replace(/\s\(\d+\)$/, "")
    .toLowerCase();
}

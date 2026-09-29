import type { OrganizationsIntegration } from "@/api-client";

import { hostedGitHubAppSlug } from "@/lib/hostedGitHubInstall";

const INSTALLATIONS_URL = "https://github.com/settings/installations";
const APP_INSTALLATION_URL_PROPERTY = "appInstallationURL";

function stringValue(source: Record<string, unknown> | undefined, key: string): string {
  const value = source?.[key];
  return typeof value === "string" ? value : "";
}

/**
 * The page on GitHub where the app installation is managed, for example to give
 * the app access to more repositories.
 *
 * Prefer `/apps/{slug}/installations/{id}`. That page belongs to the app, not
 * to one GitHub login. `/settings/installations/{id}` 404s when the user is
 * signed in to a different account. Older integrations without a slug still
 * use the stored property or the settings path.
 */
export function githubInstallationUrl(integration: OrganizationsIntegration | null | undefined): string {
  const metadata = integration?.status?.metadata;
  const installationId = stringValue(metadata, "installationId");
  const slug = hostedGitHubAppSlug(metadata);
  if (slug && installationId) {
    return `https://github.com/apps/${encodeURIComponent(slug)}/installations/${encodeURIComponent(installationId)}`;
  }

  const property = integration?.status?.properties?.find((entry) => entry.name === APP_INSTALLATION_URL_PROPERTY);
  if (property?.value) {
    return property.value;
  }

  if (!installationId) {
    return INSTALLATIONS_URL;
  }

  const organization = stringValue(integration?.spec?.configuration, "organization");
  if (organization) {
    return `https://github.com/organizations/${organization}/settings/installations/${installationId}`;
  }

  return `${INSTALLATIONS_URL}/${installationId}`;
}

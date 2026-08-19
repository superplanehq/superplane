import type { OrganizationsIntegration } from "@/api-client";

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
 * The GitHub App setup keeps that page in a property. Integrations from the
 * older setup have no properties, so the page is rebuilt from the installation
 * metadata. Organizations and personal accounts use different paths, and only
 * an installation that is bound to an organization has an organization in its
 * configuration.
 */
export function githubInstallationUrl(integration: OrganizationsIntegration | null | undefined): string {
  const property = integration?.status?.properties?.find((entry) => entry.name === APP_INSTALLATION_URL_PROPERTY);
  if (property?.value) {
    return property.value;
  }

  const installationId = stringValue(integration?.status?.metadata, "installationId");
  if (!installationId) {
    return INSTALLATIONS_URL;
  }

  const organization = stringValue(integration?.spec?.configuration, "organization");
  if (organization) {
    return `https://github.com/organizations/${organization}/settings/installations/${installationId}`;
  }

  return `${INSTALLATIONS_URL}/${installationId}`;
}

import type { ComponentType } from "react";

import type { FactorySettingsSection } from "./settingsNavItems";
import { WorkspaceIntegrationsPage } from "./settingsPrototype/WorkspaceIntegrationsPage";
import { WorkspaceMembersPage } from "./settingsPrototype/WorkspaceMembersPage";
import { WorkspaceModelsPage } from "./settingsPrototype/WorkspaceModelsPage";
import { WorkspaceRepositoriesPage } from "./settingsPrototype/WorkspaceRepositoriesPage";
import { WorkspaceSecretsPage } from "./settingsPrototype/WorkspaceSecretsPage";

export const STORYBOOK_FACTORY_SETTINGS_SECTIONS = [
  "repositories",
  "models",
  "members",
  "integrations",
  "secrets",
] as const;

export type StorybookFactorySettingsSection = (typeof STORYBOOK_FACTORY_SETTINGS_SECTIONS)[number];

export type FactorySettingsPageOverrides = Partial<Record<FactorySettingsSection, ComponentType>>;

export const STORYBOOK_FACTORY_SETTINGS_PAGES: Pick<FactorySettingsPageOverrides, StorybookFactorySettingsSection> = {
  repositories: WorkspaceRepositoriesPage,
  models: WorkspaceModelsPage,
  members: WorkspaceMembersPage,
  integrations: WorkspaceIntegrationsPage,
  secrets: WorkspaceSecretsPage,
};

export function resolveFactorySettingsPage(
  section: FactorySettingsSection,
  overrides?: FactorySettingsPageOverrides,
): ComponentType | undefined {
  if (section === "general") {
    return undefined;
  }
  return overrides?.[section];
}
